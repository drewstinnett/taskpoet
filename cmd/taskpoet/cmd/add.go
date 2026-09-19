package cmd

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/log"
	"github.com/drewstinnett/taskpoet/v2/taskpoet"
	"github.com/spf13/cobra"
)

// taskWithCmd builds a new task from the arguments and flags of the add command
func taskWithCmd(p *taskpoet.Poet, cmd *cobra.Command, args []string) (*taskpoet.Task, error) {
	priority, err := taskpoet.ParsePriority(mustGetCmd[string](cmd, "priority"))
	if err != nil {
		return nil, err
	}
	opts := []taskpoet.TaskOption{
		taskpoet.WithEffortImpact(taskpoet.EffortImpact(mustGetCmd[uint](cmd, "effort-impact"))),
		taskpoet.WithTags(mustGetCmd[[]string](cmd, "tag")),
		taskpoet.WithProject(mustGetCmd[string](cmd, "project")),
		taskpoet.WithPriority(priority),
	}

	cal := taskpoet.NewCalendar()
	for _, d := range []struct {
		flag string
		opt  func(*time.Time) taskpoet.TaskOption
	}{
		{"due", taskpoet.WithDue},
		{"wait", taskpoet.WithWait},
		{"scheduled", taskpoet.WithScheduled},
		{"until", taskpoet.WithUntil},
	} {
		if in := mustGetCmd[string](cmd, d.flag); in != "" {
			when, err := cal.Date(in)
			if err != nil {
				return nil, fmt.Errorf("--%v: %w", d.flag, err)
			}
			opts = append(opts, d.opt(when))
		}
	}

	if deps := mustGetCmd[[]string](cmd, "depends"); len(deps) > 0 {
		ids := make([]string, len(deps))
		for i, d := range deps {
			dep, err := p.Store.GetByPrefix(d, taskpoet.StatusPending)
			if err != nil {
				return nil, fmt.Errorf("--depends: %w", err)
			}
			ids[i] = dep.UUID
		}
		opts = append(opts, taskpoet.WithDepends(ids))
	}

	recur := mustGetCmd[string](cmd, "recur")
	switch {
	case recur != "":
		if _, err := taskpoet.ParseRecurrence(recur); err != nil {
			return nil, fmt.Errorf("--recur: %w", err)
		}
		opts = append(opts, taskpoet.WithRecur(recur))
	case mustGetCmd[bool](cmd, "chained"):
		return nil, errors.New("--chained only makes sense together with --recur")
	}

	task, err := taskpoet.NewTask(strings.Join(args, " "), opts...)
	if err != nil {
		return nil, err
	}
	if mustGetCmd[bool](cmd, "chained") {
		task.RType = taskpoet.RTypeChained
	}
	return task, nil
}

// newAddCmd represents the add command
func newAddCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "add DESCRIPTION",
		Short:   "Add a new task",
		Args:    cobra.MinimumNArgs(1),
		Aliases: []string{"a"},
		Example: `Add a new task by giving the description as an argument:
$ taskpoet add "Learn a new skill"

For us lazy folks, you can also leave the quotes out:
$ taskpoet add Learn a new skill

Set an Effort/Impact to a new task:
$ taskpoet add --effort-impact 2 Rebuild all the remote servers

Put it in a project, with a priority and a due date:
$ taskpoet add --project home.garden --priority H --due friday Plant the tomatoes

Make it come back every week, starting friday:
$ taskpoet add --recur weekly --due friday Water the plants`,
		Long:              `Add new task`,
		ValidArgsFunction: noComplete,
		RunE: func(cmd *cobra.Command, args []string) error {
			p := mustPoet()
			task, err := taskWithCmd(p, cmd, args)
			if err != nil {
				return err
			}
			if err := p.Add(task); err != nil {
				return err
			}
			log.Info("Added task", "description", task.Description, "id", task.ShortID())
			if task.Status == taskpoet.StatusRecurring {
				// Make the first instance right away
				rep, err := p.SpawnRecurring(false)
				if err != nil {
					return err
				}
				log.Info("Created recurring tasks", "count", len(rep.Created))
			}
			return nil
		},
	}
	return bindAdd(cmd)
}

func bindAdd(cmd *cobra.Command) *cobra.Command {
	cmd.Flags().UintP("effort-impact", "e", 0, "Effort/Impact Score Assessment. See Help for more info")
	checkErr(cmd.RegisterFlagCompletionFunc("effort-impact", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{
			"0\tUndefined (Default)",
			"1\tLow Effort, High Impact (Sweet Spot)",
			"2\t High Effort, High Impact (Homework)",
			"3\tLow Effort, Low Impact (Busywork)",
			"4\tHigh Effort, Low Impact (Charity)",
		}, cobra.ShellCompDirectiveNoFileComp
	}))

	cmd.Flags().StringSliceP("depends", "D", []string{}, "IDs of tasks that must be done before this one")
	checkErr(cmd.RegisterFlagCompletionFunc("depends", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return mustPoet().CompleteIDs(toComplete, taskpoet.StatusPending), cobra.ShellCompDirectiveNoFileComp
	}))

	cmd.Flags().StringP("project", "P", "", "Project for this task, like 'home' or 'work.reports'")
	cmd.Flags().String("priority", "", "Priority: H, M or L")
	checkErr(cmd.RegisterFlagCompletionFunc("priority", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"H\tHigh", "M\tMedium", "L\tLow"}, cobra.ShellCompDirectiveNoFileComp
	}))
	cmd.Flags().StringP("due", "d", "", "How long before this is due?")
	cmd.Flags().StringP("wait", "w", "", "Wait until given duration to actually show up as pending")
	cmd.Flags().String("scheduled", "", "When you plan to start working on this")
	cmd.Flags().String("until", "", "The task expires and is removed after this time")
	cmd.Flags().StringSliceP("tag", "t", []string{}, "Tags to include in this task")
	cmd.Flags().String("recur", "", "Make this a recurring task, like daily, weekdays, weekly, monthly, 3d, 2w. Needs --due")
	cmd.Flags().Bool("chained", false, "With --recur: the next one is due a period after you finish this one, not on a fixed schedule")
	return cmd
}
