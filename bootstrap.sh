#!/bin/bash
# Dexter Bootstrap Script
# One-command setup for the entire Dexter development environment

set -e  # Exit on error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Detect package manager
detect_package_manager() {
    if command_exists yay; then
        echo "yay"
    elif command_exists pacman; then
        echo "pacman"
    elif command_exists apt-get; then
        echo "apt"
    elif command_exists dnf; then
        echo "dnf"
    else
        echo "unknown"
    fi
}

# Install system packages
install_system_packages() {
    local pkg_manager="$1"

    log_info "Installing system dependencies..."

    case "$pkg_manager" in
        yay)
            log_info "Using yay (Arch Linux AUR helper)"
            yay -S --needed --noconfirm git go python python-pip redis
            ;;
        pacman)
            log_info "Using pacman (Arch Linux)"
            sudo pacman -S --needed --noconfirm git go python python-pip redis
            ;;
        apt)
            log_info "Using apt (Debian/Ubuntu)"
            sudo apt-get update
            sudo apt-get install -y git golang-go python3 python3-pip python3-venv redis-server
            ;;
        dnf)
            log_info "Using dnf (Fedora/RHEL)"
            sudo dnf install -y git golang python3 python3-pip redis
            ;;
        *)
            log_error "Unsupported package manager. Please install manually:"
            log_error "  - git"
            log_error "  - go (1.20+)"
            log_error "  - python3 (3.10+)"
            log_error "  - python3-pip"
            log_error "  - python3-venv (if using Debian/Ubuntu)"
            log_error "  - redis"
            exit 1
            ;;
    esac

    log_success "System packages installed"
}

# Verify required tools
verify_tools() {
    log_info "Verifying required tools..."

    local missing_tools=()

    if ! command_exists git; then
        missing_tools+=("git")
    fi

    if ! command_exists go; then
        missing_tools+=("go")
    fi

    if ! command_exists python3; then
        missing_tools+=("python3")
    fi

    if [ ${#missing_tools[@]} -gt 0 ]; then
        log_error "Missing required tools: ${missing_tools[*]}"
        return 1
    fi

    log_success "All required tools present"

    # Show versions
    log_info "Tool versions:"
    git --version
    go version
    python3 --version
}

# Setup SSH for GitHub
setup_github_ssh() {
    log_info "Checking GitHub SSH access..."

    if ssh -T git@github.com 2>&1 | grep -q "successfully authenticated"; then
        log_success "GitHub SSH access verified"
        return 0
    fi

    log_warning "GitHub SSH not configured"
    log_info "Please set up SSH keys for GitHub:"
    log_info "  1. Generate SSH key: ssh-keygen -t ed25519 -C \"your_email@example.com\""
    log_info "  2. Add to ssh-agent: eval \"\$(ssh-agent -s)\" && ssh-add ~/.ssh/id_ed25519"
    log_info "  3. Add to GitHub: cat ~/.ssh/id_ed25519.pub (copy and add to github.com/settings/keys)"
    log_info ""
    read -p "Press Enter after setting up SSH keys, or Ctrl+C to exit..."

    # Verify again
    if ssh -T git@github.com 2>&1 | grep -q "successfully authenticated"; then
        log_success "GitHub SSH access verified"
    else
        log_error "GitHub SSH still not working. Please check your setup."
        exit 1
    fi
}

# Setup directory structure
setup_directories() {
    log_info "Setting up directory structure..."

    mkdir -p ~/Dexter/{config,models,bin}
    mkdir -p ~/EasterCompany

    log_success "Directory structure created"
}

# Clone dex-cli repository
clone_dex_cli() {
    log_info "Cloning dex-cli repository..."

    if [ -d ~/Dexter/dex-cli/.git ]; then
        log_info "dex-cli already exists, pulling latest..."
        cd ~/Dexter/dex-cli
        git pull --ff-only || log_warning "Could not pull latest changes"
    else
        git clone git@github.com:EasterCompany/dex-cli.git ~/Dexter/dex-cli
    fi

    log_success "dex-cli repository ready"
}

# Build and install dex CLI
install_dex_cli() {
    log_info "Building and installing dex CLI..."

    cd ~/Dexter/dex-cli
    go build -o dex
    cp dex ~/Dexter/bin/dex
    chmod +x ~/Dexter/bin/dex

    log_success "dex CLI installed to ~/Dexter/bin/dex"
}

# Setup config files
setup_config() {
    log_info "Setting up configuration files..."

    # Check if config files exist
    if [ ! -f ~/Dexter/config/service-map.json ]; then
        log_info "Creating default service-map.json..."

        # Create a minimal service-map.json if it doesn't exist
        cat > ~/Dexter/config/service-map.json <<'EOF'
{
  "_doc": "This config file is the ultimate source of truth for mapping services from this machines perspective.",
  "service_types": [
    {
      "type": "fe",
      "label": "Frontend Application",
      "description": "Usually a HTML/CSS/JS/TS application hosted on github.com and served via github.io, but these services also reserve a local port for admin and development builds.",
      "min_port": 8000,
      "max_port": 8099
    },
    {
      "type": "cs",
      "label": "Core Service",
      "description": "Foundational resource intensive services that lay at the heart of a network.",
      "min_port": 8100,
      "max_port": 8199
    },
    {
      "type": "be",
      "label": "Backend Service",
      "description": "Backend services which can usually drop off the network and come back online without effecting anything.",
      "min_port": 8200,
      "max_port": 8299
    },
    {
      "type": "th",
      "label": "Third party integrations",
      "description": "Usually a service providing dexter with an interface to third party application (ie; discord, slack, teams ... etc ...).",
      "min_port": 8300,
      "max_port": 8399
    },
    {
      "type": "os",
      "label": "Other Service",
      "description": "This is a reserved label for other services with may or may not be crucial to the operation of the system (ie; redis local instance, redis cloud instance ...  etc ...).",
      "min_port": -1,
      "max_port": -1
    }
  ],
  "services": {
    "fe": [
      {
        "id": "easter.company",
        "source": "~/EasterCompany/easter.company",
        "repo": "git@github.com:eastercompany/eastercompany.github.io",
        "addr": "https://easter.company",
        "socket": "wss://easter.company"
      }
    ],
    "cs": [
      {
        "id": "dex-event-service",
        "source": "~/EasterCompany/dex-event-service",
        "repo": "git@github.com:eastercompany/dex-event-service",
        "addr": "http://127.0.0.1:8100/",
        "socket": "ws://127.0.0.1:8100/"
      },
      {
        "id": "dex-model-service",
        "source": "~/EasterCompany/dex-model-service",
        "repo": "git@github.com:eastercompany/dex-model-service",
        "addr": "http://127.0.0.1:8101/",
        "socket": "ws://127.0.0.1:8101/"
      }
    ],
    "be": [
      {
        "id": "dex-chat-service",
        "source": "~/EasterCompany/dex-chat-service",
        "repo": "git@github.com:eastercompany/dex-chat-service",
        "addr": "http://127.0.0.1:8200/",
        "socket": "ws://127.0.0.1:8200/"
      },
      {
        "id": "dex-stt-service",
        "source": "~/EasterCompany/dex-stt-service",
        "repo": "git@github.com:eastercompany/dex-stt-service",
        "addr": "http://127.0.0.1:8201/",
        "socket": "ws://127.0.0.1:8201/"
      },
      {
        "id": "dex-tts-service",
        "source": "~/EasterCompany/dex-tts-service",
        "repo": "git@github.com:eastercompany/dex-tts-service",
        "addr": "http://127.0.0.1:8202/",
        "socket": "ws://127.0.0.1:8202/"
      },
      {
        "id": "dex-web-service",
        "source": "~/EasterCompany/dex-web-service",
        "repo": "git@github.com:eastercompany/dex-web-service",
        "addr": "http://127.0.0.1:8203/",
        "socket": "ws://127.0.0.1:8203/"
      }
    ],
    "th": [
      {
        "id": "dex-discord-service",
        "source": "~/EasterCompany/dex-discord-service",
        "repo": "git@github.com:eastercompany/dex-discord-service",
        "addr": "http://127.0.0.1:8300/",
        "socket": "ws://127.0.0.1:8300/"
      }
    ],
    "os": [
      {
        "id": "redis-cache",
        "source": "",
        "repo": "",
        "addr": "http://127.0.0.1:6379/",
        "socket": "ws://127.0.0.1:6379/"
      }
    ]
  }
}
EOF
        log_success "Created default service-map.json"
    else
        log_info "service-map.json already exists"
    fi

    # Create options.json template if it doesn't exist
    if [ ! -f ~/Dexter/config/options.json ]; then
        log_warning "options.json not found"
        log_info "Please create ~/Dexter/config/options.json with your credentials"
        log_info "See ~/Dexter/config/options.json.default for template"
    fi
}

# Setup Python virtual environment
setup_python_venv() {
    log_info "Setting up Python virtual environment..."

    if [ -d ~/Dexter/python ]; then
        log_info "Python virtual environment already exists"
    else
        python3 -m venv ~/Dexter/python
        log_success "Python virtual environment created"
    fi

    # Activate and upgrade pip
    source ~/Dexter/python/bin/activate
    pip install --upgrade pip

    log_success "Python virtual environment ready"
}

# Clone all service repositories
clone_all_services() {
    log_info "Cloning all service repositories..."

    ~/Dexter/bin/dex pull

    log_success "All services cloned/updated"
}

# Setup shell PATH
setup_shell_path() {
    log_info "Setting up shell PATH..."

    local shell_rc=""
    if [ -n "$BASH_VERSION" ]; then
        shell_rc="$HOME/.bashrc"
    elif [ -n "$ZSH_VERSION" ]; then
        shell_rc="$HOME/.zshrc"
    else
        log_warning "Unknown shell, please manually add ~/Dexter/bin to PATH"
        return
    fi

    # Check if already in PATH
    if grep -q 'export PATH="$HOME/Dexter/bin:$PATH"' "$shell_rc" 2>/dev/null; then
        log_info "PATH already configured in $shell_rc"
    else
        echo '' >> "$shell_rc"
        echo '# Dexter CLI' >> "$shell_rc"
        echo 'export PATH="$HOME/Dexter/bin:$PATH"' >> "$shell_rc"
        log_success "Added ~/Dexter/bin to PATH in $shell_rc"
        log_warning "Run 'source $shell_rc' or restart your terminal to use 'dex' command"
    fi
}

# Main installation process
main() {
    echo ""
    echo "╔══════════════════════════════════════════════════════════╗"
    echo "║                                                          ║"
    echo "║              Dexter Bootstrap Installation               ║"
    echo "║                                                          ║"
    echo "╚══════════════════════════════════════════════════════════╝"
    echo ""

    log_info "Starting Dexter environment setup..."
    echo ""

    # Detect package manager
    PKG_MANAGER=$(detect_package_manager)
    log_info "Detected package manager: $PKG_MANAGER"

    # Install system packages
    if [ "$PKG_MANAGER" != "unknown" ]; then
        install_system_packages "$PKG_MANAGER"
    fi

    # Verify tools
    verify_tools
    echo ""

    # Setup GitHub SSH
    setup_github_ssh
    echo ""

    # Setup directories
    setup_directories
    echo ""

    # Clone dex-cli
    clone_dex_cli
    echo ""

    # Install dex CLI
    install_dex_cli
    echo ""

    # Setup config
    setup_config
    echo ""

    # Setup Python venv
    setup_python_venv
    echo ""

    # Clone all services
    clone_all_services
    echo ""

    # Setup shell PATH
    setup_shell_path
    echo ""

    # Final success message
    echo ""
    echo "╔══════════════════════════════════════════════════════════╗"
    echo "║                                                          ║"
    echo "║           ✓ Dexter Installation Complete!               ║"
    echo "║                                                          ║"
    echo "╚══════════════════════════════════════════════════════════╝"
    echo ""
    log_success "Environment ready at:"
    log_info "  ~/Dexter/       - Installation root"
    log_info "  ~/EasterCompany/ - Source code repositories"
    echo ""
    log_info "Next steps:"
    log_info "  1. Configure ~/Dexter/config/options.json with your credentials"
    log_info "  2. Run: source ~/.bashrc (or ~/.zshrc)"
    log_info "  3. Test: dex help"
    log_info "  4. Update repos: dex pull"
    echo ""
}

# Run main installation
main "$@"
