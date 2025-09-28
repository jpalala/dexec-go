# dexec: Docker/Docker Compose Alias Utility

Usage:
  # 1. Default/Exec Mode (Interactive Shell)
  dexec <container_tag> <command> [args...]  # e.g., dexec webapp sh

  # 2. Alias Mode (Docker Compose Shortcuts)
  dexec ps                      # -> docker ps
  dexec do up                      # -> docker compose up -d
  dexec do logs                    # -> docker compose logs -f
  dexec do rebuild                 # -> docker compose up -d --build

  # 3. Alias Mode (Docker Image Shortcuts)
  dexec do images                  # -> docker image ls
  dexec do rmi <tag/id>            # -> docker image rm -f <tag/id>
  dexec do rmi d                   # -> docker image prune -f (dangling)
  dexec do rmi a                   # -> docker image prune -a -f (all unused)
