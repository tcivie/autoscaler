package poller

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
)

// RepoSource lists the repos Poller should watch.
type RepoSource interface {
	ActiveRepos() ([]Repo, error)
}

type Repo struct {
	ID       int64
	CloneURL string
	Branch   string
	FullName string
}

// PipelineTrigger asks the upstream forge runner to start a build.
type PipelineTrigger interface {
	Trigger(repoID int64, branch string) error
}

// ShaStore persists last-seen SHA per repo so the Poller is stateless.
type ShaStore interface {
	Get(key string) (string, error)
	Set(key, value string) error
}

// RemoteHead resolves the current SHA of a branch in a remote git repo.
type RemoteHead interface {
	Resolve(cloneURL, branch string) (string, error)
}

// GitLsRemote shells out to `git ls-remote` to fetch a branch SHA.
type GitLsRemote struct {
	Token string
}

func (g *GitLsRemote) Resolve(cloneURL, branch string) (string, error) {
	url := cloneURL
	if g.Token != "" {
		url = injectToken(cloneURL, g.Token)
	}
	cmd := exec.Command("git", "ls-remote", "--heads", url, branch)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("ls-remote: %w", err)
	}
	fields := strings.Fields(string(out))
	if len(fields) < 1 {
		return "", fmt.Errorf("ls-remote: empty response")
	}
	return fields[0], nil
}

func injectToken(cloneURL, token string) string {
	if !strings.HasPrefix(cloneURL, "https://") {
		return cloneURL
	}
	return "https://x-access-token:" + token + "@" + strings.TrimPrefix(cloneURL, "https://")
}

// Poller drives push-to-main detection without webhooks: every interval it
// asks each repo's remote for its branch HEAD and triggers a Woodpecker
// pipeline when the SHA moves.
type Poller struct {
	repos    RepoSource
	heads    RemoteHead
	trigger  PipelineTrigger
	store    ShaStore
	interval time.Duration
}

type PollerOption func(*Poller)

func PollerWithInterval(d time.Duration) PollerOption {
	return func(p *Poller) { p.interval = d }
}

func NewPoller(repos RepoSource, heads RemoteHead, trigger PipelineTrigger, store ShaStore, opts ...PollerOption) *Poller {
	p := &Poller{
		repos:    repos,
		heads:    heads,
		trigger:  trigger,
		store:    store,
		interval: 60 * time.Second,
	}
	for _, o := range opts {
		o(p)
	}
	return p
}

func (p *Poller) Run(ctx context.Context) error {
	t := time.NewTicker(p.interval)
	defer t.Stop()

	p.Tick()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-t.C:
			p.Tick()
		}
	}
}

func (p *Poller) Tick() {
	repos, err := p.repos.ActiveRepos()
	if err != nil {
		log.Error().Err(err).Msg("poller: list repos failed")
		return
	}
	for _, r := range repos {
		p.tickRepo(r)
	}
}

func (p *Poller) tickRepo(r Repo) {
	head, err := p.heads.Resolve(r.CloneURL, r.Branch)
	if err != nil {
		log.Warn().Err(err).Str("repo", r.FullName).Msg("poller: head resolve failed")
		return
	}
	key := keyFor(r.ID)
	prev, err := p.store.Get(key)
	if err != nil {
		log.Warn().Err(err).Str("repo", r.FullName).Msg("poller: store get failed")
		return
	}
	if prev == "" {
		if err := p.store.Set(key, head); err != nil {
			log.Warn().Err(err).Str("repo", r.FullName).Msg("poller: store prime failed")
			return
		}
		log.Info().Str("repo", r.FullName).Str("sha", short(head)).Msg("poller: primed")
		return
	}
	if prev == head {
		return
	}
	if err := p.trigger.Trigger(r.ID, r.Branch); err != nil {
		log.Error().Err(err).Str("repo", r.FullName).Msg("poller: trigger failed")
		return
	}
	if err := p.store.Set(key, head); err != nil {
		log.Warn().Err(err).Str("repo", r.FullName).Msg("poller: store advance failed")
	}
	log.Info().Str("repo", r.FullName).Str("from", short(prev)).Str("to", short(head)).Msg("poller: triggered")
}

func keyFor(id int64) string { return fmt.Sprintf("%d", id) }

func short(sha string) string {
	if len(sha) > 8 {
		return sha[:8]
	}
	return sha
}
