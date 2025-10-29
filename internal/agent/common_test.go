package agent

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"charm.land/fantasy"
	"charm.land/fantasy/providers/anthropic"
	"charm.land/fantasy/providers/openai"
	"charm.land/fantasy/providers/openaicompat"
	"charm.land/fantasy/providers/openrouter"
	"github.com/charmbracelet/catwalk/pkg/catwalk"
	"github.com/eliasbui/ccl-magic/internal/agent/prompt"
	"github.com/eliasbui/ccl-magic/internal/agent/tools"
	"github.com/eliasbui/ccl-magic/internal/config"
	"github.com/eliasbui/ccl-magic/internal/csync"
	"github.com/eliasbui/ccl-magic/internal/db"
	"github.com/eliasbui/ccl-magic/internal/history"
	"github.com/eliasbui/ccl-magic/internal/lsp"
	"github.com/eliasbui/ccl-magic/internal/message"
	"github.com/eliasbui/ccl-magic/internal/permission"
	"github.com/eliasbui/ccl-magic/internal/session"
	"github.com/stretchr/testify/require"
	"gopkg.in/dnaeon/go-vcr.v4/pkg/recorder"

	_ "github.com/joho/godotenv/autoload"
)

type env struct {
	workingDir  string
	sessions    session.Service
	messages    message.Service
	permissions permission.Service
	history     history.Service
	lspClients  *csync.Map[string, *lsp.Client]
}

type builderFunc func(t *testing.T, r *recorder.Recorder) (fantasy.LanguageModel, error)

type modelPair struct {
	name       string
	largeModel builderFunc
	smallModel builderFunc
}

func anthropicBuilder(model string) builderFunc {
	return func(t *testing.T, r *recorder.Recorder) (fantasy.LanguageModel, error) {
		provider, err := anthropic.New(
			anthropic.WithAPIKey(os.Getenv("CRUSH_ANTHROPIC_API_KEY")),
			anthropic.WithHTTPClient(&http.Client{Transport: r}),
		)
		if err != nil {
			return nil, err
		}
		return provider.LanguageModel(t.Context(), model)
	}
}

func openaiBuilder(model string) builderFunc {
	return func(t *testing.T, r *recorder.Recorder) (fantasy.LanguageModel, error) {
		provider, err := openai.New(
			openai.WithAPIKey(os.Getenv("CRUSH_OPENAI_API_KEY")),
			openai.WithHTTPClient(&http.Client{Transport: r}),
		)
		if err != nil {
			return nil, err
		}
		return provider.LanguageModel(t.Context(), model)
	}
}

func openRouterBuilder(model string) builderFunc {
	return func(t *testing.T, r *recorder.Recorder) (fantasy.LanguageModel, error) {
		provider, err := openrouter.New(
			openrouter.WithAPIKey(os.Getenv("CRUSH_OPENROUTER_API_KEY")),
			openrouter.WithHTTPClient(&http.Client{Transport: r}),
		)
		if err != nil {
			return nil, err
		}
		return provider.LanguageModel(t.Context(), model)
	}
}

func zAIBuilder(model string) builderFunc {
	return func(t *testing.T, r *recorder.Recorder) (fantasy.LanguageModel, error) {
		provider, err := openaicompat.New(
			openaicompat.WithBaseURL("https://api.z.ai/api/coding/paas/v4"),
			openaicompat.WithAPIKey(os.Getenv("CRUSH_ZAI_API_KEY")),
			openaicompat.WithHTTPClient(&http.Client{Transport: r}),
		)
		if err != nil {
			return nil, err
		}
		return provider.LanguageModel(t.Context(), model)
	}
}

func testEnv(t *testing.T) env {
	testDir := filepath.Join("/tmp/crush-test/", t.Name())
	os.RemoveAll(testDir)
	err := os.MkdirAll(testDir, 0o755)
	require.NoError(t, err)
	workingDir := testDir
	conn, err := db.Connect(t.Context(), t.TempDir())
	require.NoError(t, err)
	q := db.New(conn)
	sessions := session.NewService(q)
	messages := message.NewService(q)
	permissions := permission.NewPermissionService(workingDir, true, []string{})
	history := history.NewService(q, conn)
	lspClients := csync.NewMap[string, *lsp.Client]()

	t.Cleanup(func() {
		conn.Close()
		os.RemoveAll(testDir)
	})

	return env{
		workingDir,
		sessions,
		messages,
		permissions,
		history,
		lspClients,
	}
}

func testSessionAgent(env env, large, small fantasy.LanguageModel, systemPrompt string, tools ...fantasy.AgentTool) SessionAgent {
	largeModel := Model{
		Model: large,
		CatwalkCfg: catwalk.Model{
			ContextWindow:    200000,
			DefaultMaxTokens: 10000,
		},
	}
	smallModel := Model{
		Model: small,
		CatwalkCfg: catwalk.Model{
			ContextWindow:    200000,
			DefaultMaxTokens: 10000,
		},
	}
	agent := NewSessionAgent(SessionAgentOptions{largeModel, smallModel, "", systemPrompt, false, true, env.sessions, env.messages, tools})
	return agent
}

func coderAgent(r *recorder.Recorder, env env, large, small fantasy.LanguageModel) (SessionAgent, error) {
	fixedTime := func() time.Time {
		t, _ := time.Parse("1/2/2006", "1/1/2025")
		return t
	}
	prompt, err := coderPrompt(
		prompt.WithTimeFunc(fixedTime),
		prompt.WithPlatform("linux"),
		prompt.WithWorkingDir(filepath.ToSlash(env.workingDir)),
	)
	if err != nil {
		return nil, err
	}
	cfg, err := config.Init(env.workingDir, "", false)
	if err != nil {
		return nil, err
	}

	systemPrompt, err := prompt.Build(context.TODO(), large.Provider(), large.Model(), *cfg)
	if err != nil {
		return nil, err
	}
	allTools := []fantasy.AgentTool{
		tools.NewBashTool(env.permissions, env.workingDir, cfg.Options.Attribution),
		tools.NewDownloadTool(env.permissions, env.workingDir, r.GetDefaultClient()),
		tools.NewEditTool(env.lspClients, env.permissions, env.history, env.workingDir),
		tools.NewMultiEditTool(env.lspClients, env.permissions, env.history, env.workingDir),
		tools.NewFetchTool(env.permissions, env.workingDir, r.GetDefaultClient()),
		tools.NewGlobTool(env.workingDir),
		tools.NewGrepTool(env.workingDir),
		tools.NewLsTool(env.permissions, env.workingDir, cfg.Tools.Ls),
		tools.NewSourcegraphTool(r.GetDefaultClient()),
		tools.NewViewTool(env.lspClients, env.permissions, env.workingDir),
		tools.NewWriteTool(env.lspClients, env.permissions, env.history, env.workingDir),
	}

	return testSessionAgent(env, large, small, systemPrompt, allTools...), nil
}

// createSimpleGoProject creates a simple Go project structure in the given directory.
// It creates a go.mod file and a main.go file with a basic hello world program.
func createSimpleGoProject(t *testing.T, dir string) {
	goMod := `module example.com/testproject

go 1.23
`
	err := os.WriteFile(dir+"/go.mod", []byte(goMod), 0o644)
	require.NoError(t, err)

	mainGo := `package main

import "fmt"

func main() {
	fmt.Println("Hello, World!")
}
`
	err = os.WriteFile(dir+"/main.go", []byte(mainGo), 0o644)
	require.NoError(t, err)
}
