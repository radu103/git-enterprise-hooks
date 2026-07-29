package commit

import (
	"strings"

	"github.com/radu103/git-enterprise-hooks/internal/domain"
)

func Render(template string, ctx domain.CommitMessageContext) string {
	replacer := strings.NewReplacer(
		"{task_key}", ctx.TaskKey,
		"{task_title}", ctx.TaskTitle,
		"{task_epic}", ctx.TaskEpic,
		"{summary}", ctx.Summary,
	)
	return replacer.Replace(template)
}
