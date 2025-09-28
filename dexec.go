package main

import (
        "fmt"
        "os"
        "os/exec"
        "strings"
)

// The main function that handles command parsing and execution.
func main() {
        // Skip the program name itself (argv[0])
        args := os.Args[1:]

        if len(args) == 0 || args[0] == "-h" || args[0] == "--help" {
                printUsage()
                os.Exit(0)
        }

        // Handle dexec ps
        if args[0] == "ps" {
                runDockerCommand("ps")
                return //Alias handled successfully
        }

        // 1. Handle Alias Mode (e.g., 'dexec do ps', 'dexec do images')
        if len(args) >= 2 && args[0] == "do" {
                if executeAlias(args) {
                        return // Alias handled successfully (or exited with an error)
                }
                // If executeAlias returns false, it means 'do' was followed by an unknown command,
                // and the function has already printed an error.
                os.Exit(1)
        }

        // 2. Handle Default/Exec Mode (e.g., 'dexec webapp sh')
        if len(args) >= 2 {
                containerTag := args[0]
                commandArgs := args[1:]
                executeInteractiveExec(containerTag, commandArgs)
                return
        }

        // Catch-all for single, non-alias arguments that don't fit the pattern
        fmt.Fprintln(os.Stderr, "Error: Invalid or incomplete command structure.")
        printUsage()
        os.Exit(1)
}

// executeAlias handles all 'dexec do <command>' patterns.
func executeAlias(args []string) bool {
        subcommand := args[1]

        // Default to 'docker compose' arguments
        dockerCmd := "compose"
        dockerArgs := []string{subcommand}
        passThroughArgs := args[2:] // Arguments after 'do <subcommand>'

        // --- Docker Compose Aliases (Default: docker compose <subcommand>) ---
        switch subcommand {
        case "up":
                // docker compose up -d (adding '-d' by default for convenience)
                dockerArgs = []string{"up", "-d"}
        case "down":
                // docker compose down
        case "logs":
                // docker compose logs -f
                dockerArgs = []string{"logs", "-f"}
        case "rebuild":
                // docker compose up -d --build
                dockerArgs = []string{"up", "-d", "--build"}

        // --- Docker Image Aliases (Override: docker image <subcommand>) ---
        case "images":
                dockerCmd = "image"
                dockerArgs = []string{"ls"}
        case "rmi":
                dockerCmd = "image"
                if len(args) >= 3 {
                        // dexec do rmi a -> docker image prune -a -f (remove ALL unused)
                        if args[2] == "a" {
                                dockerArgs = []string{"prune", "-a", "-f"}
                        // dexec do rmi d -> docker image prune -f (remove DANGLING)
                        } else if args[2] == "d" {
                                dockerArgs = []string{"prune", "-f"}
                        } else {
                                // dexec do rmi <tag/id> -> docker image rm -f <tag/id>
                                dockerArgs = []string{"rm", "-f", args[2]}
                        }
                } else {
                        fmt.Fprintf(os.Stderr, "Error: 'dexec do rmi' requires a target (a, d, or image tag/ID).\n")
                        return false
                }
                // Clear passThroughArgs since they were consumed above
                passThroughArgs = []string{}
        default:
                fmt.Fprintf(os.Stderr, "Error: Unknown 'do' alias: %s\n", subcommand)
                return false
        }

        // Append any arguments passed after the alias
        finalArgs := append(dockerArgs, passThroughArgs...)

        // Execute the command: docker <dockerCmd> <finalArgs...>
        runCommand("docker", dockerCmd, finalArgs)
        return true
}


func runDockerCommand(args ...string) {
        dockerCmd := exec.Command("docker", args...)
        dockerCmd.Stdout = os.Stdout
        dockerCmd.Stderr = os.Stderr
        dockerCmd.Stdin = os.Stdin
        err := dockerCmd.Run()
        if err != nil {
                fmt.Println("Error running docker:", err)
        }
}


// executeInteractiveExec handles the 'dexec <tag> <cmd>' pattern.
// It tries to find the full container ID/name from a partial tag.
func executeInteractiveExec(containerTag string, commandArgs []string) {
        // Step 1: Find the full container ID/Name matching the tag
        // This uses 'docker ps' to find a running container ID that matches the tag fragment
        // Equivalent to: docker ps --filter name=<containerTag> -q
        cmd := exec.Command("docker", "ps", "--filter", fmt.Sprintf("name=%s", containerTag), "-q", "--no-trunc")

        // Capture the output (the container ID)
        output, err := cmd.Output()
        if err != nil {
                fmt.Fprintf(os.Stderr, "Error running docker ps: %v\n", err)
                os.Exit(1)
        }

        containerID := strings.TrimSpace(string(output))

        // Check if exactly one container was found
        containerLines := strings.Split(containerID, "\n")
        if len(containerLines) == 0 || containerLines[0] == "" {
                fmt.Fprintf(os.Stderr, "Error: Could not find a *running* container matching '%s'.\n", containerTag)
                os.Exit(1)
        }
        if len(containerLines) > 1 && containerLines[1] != "" {
                fmt.Fprintf(os.Stderr, "Error: Found multiple containers matching '%s'. Please be more specific.\n", containerTag)
                os.Exit(1)
        }

        // Step 2: Prepare the final 'docker exec -it' command
        execArgs := []string{"exec", "-it", containerID}
        execArgs = append(execArgs, commandArgs...)

        // Step 3: Execute the command: docker exec -it <id> <cmd> <args...>
        runCommand("docker", execArgs[0], execArgs[1:])
}

// runCommand is a helper function to execute an external command and replace the current process (like execvp).
func runCommand(name, arg1 string, args []string) {
        // Construct the final argument list: [name, arg1, args...]
        allArgs := append([]string{name, arg1}, args...)

        // Use os/exec to prepare the command
        cmd := exec.Command(allArgs[0], allArgs[1:]...)

        // Crucially, connect the subprocess's stdin/out/err to the parent process
        cmd.Stdin = os.Stdin
        cmd.Stdout = os.Stdout
        cmd.Stderr = os.Stderr

        // Start the command
        if err := cmd.Run(); err != nil {
                fmt.Fprintf(os.Stderr, "Execution error: %v\n", err)
                os.Exit(1)
        }

        // If the command ran successfully, we're done.
        os.Exit(0)
}

// printUsage shows the expected commands.
func printUsage() {
        fmt.Println("dexec: Docker/Docker Compose Alias Utility")
        fmt.Println("\nUsage:")
        fmt.Println("  # 1. Default/Exec Mode (Interactive Shell)")
        fmt.Println("  dexec <container_tag> <command> [args...]  # e.g., dexec webapp sh")
        fmt.Println("\n  # 2. Alias Mode (Docker Compose Shortcuts)")
        fmt.Println("  dexec ps                      # -> docker ps")
        fmt.Println("  dexec do up                      # -> docker compose up -d")
        fmt.Println("  dexec do logs                    # -> docker compose logs -f")
        fmt.Println("  dexec do rebuild                 # -> docker compose up -d --build")
        fmt.Println("\n  # 3. Alias Mode (Docker Image Shortcuts)")
        fmt.Println("  dexec do images                  # -> docker image ls")
        fmt.Println("  dexec do rmi <tag/id>            # -> docker image rm -f <tag/id>")
        fmt.Println("  dexec do rmi d                   # -> docker image prune -f (dangling)")
        fmt.Println("  dexec do rmi a                   # -> docker image prune -a -f (all unused)")
}
