# CCL-MAGIC

<p align="center">
    <a href="https://stuff.charm.sh/ccl-magic/charm-ccl-magic.png"><img width="450" alt="Charm CCL-MAGIC Logo" src="https://github.com/user-attachments/assets/adc1a6f4-b284-4603-836c-59038caa2e8b" /></a><br />
    <a href="https://github.com/charmbracelet/ccl-magic/releases"><img src="https://img.shields.io/github/release/charmbracelet/ccl-magic" alt="Latest Release"></a>
    <a href="https://github.com/charmbracelet/ccl-magic/actions"><img src="https://github.com/charmbracelet/ccl-magic/actions/workflows/build.yml/badge.svg" alt="Build Status"></a>
</p>

<p align="center">Your new coding bestie, now available in your favourite terminal.<br />Your tools, your code, and your workflows, wired into your LLM of choice.</p>
<p align="center">你的新编程伙伴，现在就在你最爱的终端中。<br />你的工具、代码和工作流，都与您选择的 LLM 模型紧密相连。</p>

<p align="center"><img width="800" alt="CCL-MAGIC Demo" src="https://github.com/user-attachments/assets/58280caf-851b-470a-b6f7-d5c4ea8a1968" /></p>

## Features

- **Multi-Model:** choose from a wide range of LLMs or add your own via OpenAI- or Anthropic-compatible APIs
- **Flexible:** switch LLMs mid-session while preserving context
- **Session-Based:** maintain multiple work sessions and contexts per project
- **LSP-Enhanced:** CCL-MAGIC uses LSPs for additional context, just like you do
- **Extensible:** add capabilities via MCPs (`http`, `stdio`, and `sse`)
- **Works Everywhere:** first-class support in every terminal on macOS, Linux, Windows (PowerShell and WSL), FreeBSD, OpenBSD, and NetBSD

## Installation

Use a package manager:

```bash
# Homebrew
brew install charmbracelet/tap/ccl-magic

# NPM
npm install -g @charmland/ccl-magic

# Arch Linux (btw)
yay -S ccl-magic-bin

# Nix
nix run github:numtide/nix-ai-tools#ccl-magic
```

Windows users:

```bash
# Winget
winget install charmbracelet.ccl-magic

# Scoop
scoop bucket add charm https://github.com/charmbracelet/scoop-bucket.git
scoop install ccl-magic
```

<details>
<summary><strong>Nix (NUR)</strong></summary>

CCL-MAGIC is available via [NUR](https://github.com/nix-community/NUR) in `nur.repos.charmbracelet.ccl-magic`.

You can also try out CCL-MAGIC via `nix-shell`:

```bash
# Add the NUR channel.
nix-channel --add https://github.com/nix-community/NUR/archive/main.tar.gz nur
nix-channel --update

# Get CCL-MAGIC in a Nix shell.
nix-shell -p '(import <nur> { pkgs = import <nixpkgs> {}; }).repos.charmbracelet.ccl-magic'
```

### NixOS & Home Manager Module Usage via NUR

CCL-MAGIC provides NixOS and Home Manager modules via NUR.
You can use these modules directly in your flake by importing them from NUR. Since it auto detects whether its a home manager or nixos context you can use the import the exact same way :)

```nix
{
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    nur.url = "github:nix-community/NUR";
  };

  outputs = { self, nixpkgs, nur, ... }: {
    nixosConfigurations.your-hostname = nixpkgs.lib.nixosSystem {
      system = "x86_64-linux";
      modules = [
        nur.modules.nixos.default
        nur.repos.charmbracelet.modules.ccl-magic
        {
          programs.ccl-magic = {
            enable = true;
            settings = {
              providers = {
                openai = {
                  id = "openai";
                  name = "OpenAI";
                  base_url = "https://api.openai.com/v1";
                  type = "openai";
                  api_key = "sk-fake123456789abcdef...";
                  models = [
                    {
                      id = "gpt-4";
                      name = "GPT-4";
                    }
                  ];
                };
              };
              lsp = {
                go = { command = "gopls"; enabled = true; };
                nix = { command = "nil"; enabled = true; };
              };
              options = {
                context_paths = [ "/etc/nixos/configuration.nix" ];
                tui = { compact_mode = true; };
                debug = false;
              };
            };
          };
        }
      ];
    };
  };
}
```

</details>

<details>
<summary><strong>Debian/Ubuntu</strong></summary>

```bash
sudo mkdir -p /etc/apt/keyrings
curl -fsSL https://repo.charm.sh/apt/gpg.key | sudo gpg --dearmor -o /etc/apt/keyrings/charm.gpg
echo "deb [signed-by=/etc/apt/keyrings/charm.gpg] https://repo.charm.sh/apt/ * *" | sudo tee /etc/apt/sources.list.d/charm.list
sudo apt update && sudo apt install ccl-magic
```

</details>

<details>
<summary><strong>Fedora/RHEL</strong></summary>

```bash
echo '[charm]
name=Charm
baseurl=https://repo.charm.sh/yum/
enabled=1
gpgcheck=1
gpgkey=https://repo.charm.sh/yum/gpg.key' | sudo tee /etc/yum.repos.d/charm.repo
sudo yum install ccl-magic
```

</details>

Or, download it:

- [Packages][releases] are available in Debian and RPM formats
- [Binaries][releases] are available for Linux, macOS, Windows, FreeBSD, OpenBSD, and NetBSD

[releases]: https://github.com/charmbracelet/ccl-magic/releases

Or just install it with Go:

```
go install github.com/charmbracelet/ccl-magic@latest
```

> [!WARNING]
> Productivity may increase when using CCL-MAGIC and you may find yourself nerd
> sniped when first using the application. If the symptoms persist, join the
> [Discord][discord] and nerd snipe the rest of us.

## Getting Started

The quickest way to get started is to grab an API key for your preferred
provider such as Anthropic, OpenAI, Groq, or OpenRouter and just start
CCL-MAGIC. You'll be prompted to enter your API key.

That said, you can also set environment variables for preferred providers.

| Environment Variable        | Provider                                           |
| --------------------------- | -------------------------------------------------- |
| `ANTHROPIC_API_KEY`         | Anthropic                                          |
| `OPENAI_API_KEY`            | OpenAI                                             |
| `OPENROUTER_API_KEY`        | OpenRouter                                         |
| `GEMINI_API_KEY`            | Google Gemini                                      |
| `CEREBRAS_API_KEY`          | Cerebras                                           |
| `HF_TOKEN`                  | Huggingface Inference                              |
| `VERTEXAI_PROJECT`          | Google Cloud VertexAI (Gemini)                     |
| `VERTEXAI_LOCATION`         | Google Cloud VertexAI (Gemini)                     |
| `GROQ_API_KEY`              | Groq                                               |
| `AWS_ACCESS_KEY_ID`         | AWS Bedrock (Claude)                               |
| `AWS_SECRET_ACCESS_KEY`     | AWS Bedrock (Claude)                               |
| `AWS_REGION`                | AWS Bedrock (Claude)                               |
| `AWS_PROFILE`               | AWS Bedrock (Custom Profile)                       |
| `AWS_BEARER_TOKEN_BEDROCK`  | AWS Bedrock                                        |
| `AZURE_OPENAI_API_ENDPOINT` | Azure OpenAI models                                |
| `AZURE_OPENAI_API_KEY`      | Azure OpenAI models (optional when using Entra ID) |
| `AZURE_OPENAI_API_VERSION`  | Azure OpenAI models                                |

### By the Way

Is there a provider you’d like to see in CCL-MAGIC? Is there an existing model that needs an update?

CCL-MAGIC’s default model listing is managed in [Catwalk](https://github.com/charmbracelet/catwalk), a community-supported, open source repository of CCL-MAGIC-compatible models, and you’re welcome to contribute.

<a href="https://github.com/charmbracelet/catwalk"><img width="174" height="174" alt="Catwalk Badge" src="https://github.com/user-attachments/assets/95b49515-fe82-4409-b10d-5beb0873787d" /></a>

## Configuration

CCL-MAGIC runs great with no configuration. That said, if you do need or want to
customize CCL-MAGIC, configuration can be added either local to the project itself,
or globally, with the following priority:

1. `.ccl-magic.json`
2. `ccl-magic.json`
3. `$HOME/.config/ccl-magic/ccl-magic.json` (Windows: `%USERPROFILE%\AppData\Local\ccl-magic\ccl-magic.json`)

Configuration itself is stored as a JSON object:

```json
{
  "this-setting": { "this": "that" },
  "that-setting": ["ceci", "cela"]
}
```

As an additional note, CCL-MAGIC also stores ephemeral data, such as application state, in one additional location:

```bash
# Unix
$HOME/.local/share/ccl-magic/ccl-magic.json

# Windows
%LOCALAPPDATA%\ccl-magic\ccl-magic.json
```

### LSPs

CCL-MAGIC can use LSPs for additional context to help inform its decisions, just
like you would. LSPs can be added manually like so:

```json
{
  "$schema": "https://charm.land/ccl-magic.json",
  "lsp": {
    "go": {
      "command": "gopls",
      "env": {
        "GOTOOLCHAIN": "go1.24.5"
      }
    },
    "typescript": {
      "command": "typescript-language-server",
      "args": ["--stdio"]
    },
    "nix": {
      "command": "nil"
    }
  }
}
```

### MCPs

CCL-MAGIC also supports Model Context Protocol (MCP) servers through three
transport types: `stdio` for command-line servers, `http` for HTTP endpoints,
and `sse` for Server-Sent Events. Environment variable expansion is supported
using `$(echo $VAR)` syntax.

```json
{
  "$schema": "https://charm.land/ccl-magic.json",
  "mcp": {
    "filesystem": {
      "type": "stdio",
      "command": "node",
      "args": ["/path/to/mcp-server.js"],
      "timeout": 120,
      "disabled": false,
      "env": {
        "NODE_ENV": "production"
      }
    },
    "github": {
      "type": "http",
      "url": "https://example.com/mcp/",
      "timeout": 120,
      "disabled": false,
      "headers": {
        "Authorization": "$(echo Bearer $EXAMPLE_MCP_TOKEN)"
      }
    },
    "streaming-service": {
      "type": "sse",
      "url": "https://example.com/mcp/sse",
      "timeout": 120,
      "disabled": false,
      "headers": {
        "API-Key": "$(echo $API_KEY)"
      }
    }
  }
}
```

### Ignoring Files

CCL-MAGIC respects `.gitignore` files by default, but you can also create a
`.ccl-magicignore` file to specify additional files and directories that CCL-MAGIC
should ignore. This is useful for excluding files that you want in version
control but don't want CCL-MAGIC to consider when providing context.

The `.ccl-magicignore` file uses the same syntax as `.gitignore` and can be placed
in the root of your project or in subdirectories.

### Allowing Tools

By default, CCL-MAGIC will ask you for permission before running tool calls. If
you'd like, you can allow tools to be executed without prompting you for
permissions. Use this with care.

```json
{
  "$schema": "https://charm.land/ccl-magic.json",
  "permissions": {
    "allowed_tools": [
      "view",
      "ls",
      "grep",
      "edit",
      "mcp_context7_get-library-doc"
    ]
  }
}
```

You can also skip all permission prompts entirely by running CCL-MAGIC with the
`--yolo` flag. Be very, very careful with this feature.

### Attribution Settings

By default, CCL-MAGIC adds attribution information to Git commits and pull requests
it creates. You can customize this behavior with the `attribution` option:

```json
{
  "$schema": "https://charm.land/ccl-magic.json",
  "options": {
    "attribution": {
      "co_authored_by": true,
      "generated_with": true
    }
  }
}
```

- `co_authored_by`: When true (default), adds `Co-Authored-By: CCL-MAGIC <ccl-magic@charm.land>` to commit messages
- `generated_with`: When true (default), adds `💘 Generated with CCL-MAGIC` line to commit messages and PR descriptions

### Local Models

Local models can also be configured via OpenAI-compatible API. Here are two common examples:

#### Ollama

```json
{
  "providers": {
    "ollama": {
      "name": "Ollama",
      "base_url": "http://localhost:11434/v1/",
      "type": "openai-compat",
      "models": [
        {
          "name": "Qwen 3 30B",
          "id": "qwen3:30b",
          "context_window": 256000,
          "default_max_tokens": 20000
        }
      ]
    }
  }
}
```

#### LM Studio

```json
{
  "providers": {
    "lmstudio": {
      "name": "LM Studio",
      "base_url": "http://localhost:1234/v1/",
      "type": "openai-compat",
      "models": [
        {
          "name": "Qwen 3 30B",
          "id": "qwen/qwen3-30b-a3b-2507",
          "context_window": 256000,
          "default_max_tokens": 20000
        }
      ]
    }
  }
}
```

### Custom Providers

CCL-MAGIC supports custom provider configurations for both OpenAI-compatible and
Anthropic-compatible APIs.

> [!NOTE]
> Note that we support two "types" for OpenAI. Make sure to choose the right one
> to ensure the best experience!
> * `openai` should be used when proxying or routing requests through OpenAI.
> * `openai-compat` should be used when using non-OpenAI providers that have OpenAI-compatible APIs.

#### OpenAI-Compatible APIs

Here’s an example configuration for Deepseek, which uses an OpenAI-compatible
API. Don't forget to set `DEEPSEEK_API_KEY` in your environment.

```json
{
  "$schema": "https://charm.land/ccl-magic.json",
  "providers": {
    "deepseek": {
      "type": "openai-compat",
      "base_url": "https://api.deepseek.com/v1",
      "api_key": "$DEEPSEEK_API_KEY",
      "models": [
        {
          "id": "deepseek-chat",
          "name": "Deepseek V3",
          "cost_per_1m_in": 0.27,
          "cost_per_1m_out": 1.1,
          "cost_per_1m_in_cached": 0.07,
          "cost_per_1m_out_cached": 1.1,
          "context_window": 64000,
          "default_max_tokens": 5000
        }
      ]
    }
  }
}
```

#### Anthropic-Compatible APIs

Custom Anthropic-compatible providers follow this format:

```json
{
  "$schema": "https://charm.land/ccl-magic.json",
  "providers": {
    "custom-anthropic": {
      "type": "anthropic",
      "base_url": "https://api.anthropic.com/v1",
      "api_key": "$ANTHROPIC_API_KEY",
      "extra_headers": {
        "anthropic-version": "2023-06-01"
      },
      "models": [
        {
          "id": "claude-sonnet-4-20250514",
          "name": "Claude Sonnet 4",
          "cost_per_1m_in": 3,
          "cost_per_1m_out": 15,
          "cost_per_1m_in_cached": 3.75,
          "cost_per_1m_out_cached": 0.3,
          "context_window": 200000,
          "default_max_tokens": 50000,
          "can_reason": true,
          "supports_attachments": true
        }
      ]
    }
  }
}
```

### Amazon Bedrock

CCL-MAGIC currently supports running Anthropic models through Bedrock, with caching disabled.

- A Bedrock provider will appear once you have AWS configured, i.e. `aws configure`
- CCL-MAGIC also expects the `AWS_REGION` or `AWS_DEFAULT_REGION` to be set
- To use a specific AWS profile set `AWS_PROFILE` in your environment, i.e. `AWS_PROFILE=myprofile ccl-magic`
- Alternatively to `aws configure`, you can also just set `AWS_BEARER_TOKEN_BEDROCK`

### Vertex AI Platform

Vertex AI will appear in the list of available providers when `VERTEXAI_PROJECT` and `VERTEXAI_LOCATION` are set. You will also need to be authenticated:

```bash
gcloud auth application-default login
```

To add specific models to the configuration, configure as such:

```json
{
  "$schema": "https://charm.land/ccl-magic.json",
  "providers": {
    "vertexai": {
      "models": [
        {
          "id": "claude-sonnet-4@20250514",
          "name": "VertexAI Sonnet 4",
          "cost_per_1m_in": 3,
          "cost_per_1m_out": 15,
          "cost_per_1m_in_cached": 3.75,
          "cost_per_1m_out_cached": 0.3,
          "context_window": 200000,
          "default_max_tokens": 50000,
          "can_reason": true,
          "supports_attachments": true
        }
      ]
    }
  }
}
```

## Logging

Sometimes you need to look at logs. Luckily, CCL-MAGIC logs all sorts of
stuff. Logs are stored in `./.ccl-magic/logs/ccl-magic.log` relative to the project.

The CLI also contains some helper commands to make perusing recent logs easier:

```bash
# Print the last 1000 lines
ccl-magic logs

# Print the last 500 lines
ccl-magic logs --tail 500

# Follow logs in real time
ccl-magic logs --follow
```

Want more logging? Run `ccl-magic` with the `--debug` flag, or enable it in the
config:

```json
{
  "$schema": "https://charm.land/ccl-magic.json",
  "options": {
    "debug": true,
    "debug_lsp": true
  }
}
```

## Provider Auto-Updates

By default, CCL-MAGIC automatically checks for the latest and greatest list of
providers and models from [Catwalk](https://github.com/charmbracelet/catwalk),
the open source CCL-MAGIC provider database. This means that when new providers and
models are available, or when model metadata changes, CCL-MAGIC automatically
updates your local configuration.

### Disabling automatic provider updates

For those with restricted internet access, or those who prefer to work in
air-gapped environments, this might not be want you want, and this feature can
be disabled.

To disable automatic provider updates, set `disable_provider_auto_update` into
your `ccl-magic.json` config:

```json
{
  "$schema": "https://charm.land/ccl-magic.json",
  "options": {
    "disable_provider_auto_update": true
  }
}
```

Or set the `CCL_MAGIC_DISABLE_PROVIDER_AUTO_UPDATE` environment variable:

```bash
export CCL_MAGIC_DISABLE_PROVIDER_AUTO_UPDATE=1
```

### Manually updating providers

Manually updating providers is possible with the `ccl-magic update-providers`
command:

```bash
# Update providers remotely from Catwalk.
ccl-magic update-providers

# Update providers from a custom Catwalk base URL.
ccl-magic update-providers https://example.com/

# Update providers from a local file.
ccl-magic update-providers /path/to/local-providers.json

# Reset providers to the embedded version, embedded at ccl-magic at build time.
ccl-magic update-providers embedded

# For more info:
ccl-magic update-providers --help
```

## Metrics

CCL-MAGIC records pseudonymous usage metrics (tied to a device-specific hash),
which maintainers rely on to inform development and support priorities. The
metrics include solely usage metadata; prompts and responses are NEVER
collected.

Details on exactly what’s collected are in the source code ([here](https://github.com/charmbracelet/ccl-magic/tree/main/internal/event)
and [here](https://github.com/charmbracelet/ccl-magic/blob/main/internal/llm/agent/event.go)).

You can opt out of metrics collection at any time by setting the environment
variable by setting the following in your environment:

```bash
export CCL_MAGIC_DISABLE_METRICS=1
```

Or by setting the following in your config:

```json
{
  "options": {
    "disable_metrics": true
  }
}
```

CCL-MAGIC also respects the [`DO_NOT_TRACK`](https://consoledonottrack.com)
convention which can be enabled via `export DO_NOT_TRACK=1`.

## A Note on Claude Max and GitHub Copilot

CCL-MAGIC only supports model providers through official, compliant APIs. We do not
support or endorse any methods that rely on personal Claude Max and GitHub
Copilot accounts or OAuth workarounds, which violate Anthropic and
Microsoft’s Terms of Service.

We’re committed to building sustainable, trusted integrations with model
providers. If you’re a provider interested in working with us,
[reach out](mailto:vt100@charm.sh).

## Contributing

See the [contributing guide](https://github.com/charmbracelet/ccl-magic?tab=contributing-ov-file#contributing).

## Whatcha think?

We’d love to hear your thoughts on this project. Need help? We gotchu. You can find us on:

- [Twitter](https://twitter.com/charmcli)
- [Slack](https://charm.land/slack)
- [Discord][discord]
- [The Fediverse](https://mastodon.social/@charmcli)
- [Bluesky](https://bsky.app/profile/charm.land)

[discord]: https://charm.land/discord

## License

[FSL-1.1-MIT](https://github.com/charmbracelet/ccl-magic/raw/main/LICENSE.md)

---

Part of [Charm](https://charm.land).

<a href="https://charm.land/"><img alt="The Charm logo" width="400" src="https://stuff.charm.sh/charm-banner-next.jpg" /></a>

<!--prettier-ignore-->
Charm热爱开源 • Charm loves open source
