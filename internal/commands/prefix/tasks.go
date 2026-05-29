package prefix

import (
	"discord-bot/internal/commands"
	"discord-bot/internal/models"
	"fmt"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

type TasksCommand struct{}

func (c *TasksCommand) Name() string {
	return "tasks"
}

func (c *TasksCommand) Aliases() []string {
	return []string{"görevler", "gorevler", "task"}
}

func (c *TasksCommand) Description() string {
	return "Mevcut görevleri ve ilerlemenizi listeler."
}

func (c *TasksCommand) Execute(ctx *commands.Context) error {
	targetUser, _, err := resolveTargetUser(ctx)
	if err != nil {
		_, err := ctx.Session.ChannelMessageSendReply(ctx.Message.ChannelID, "❌ "+err.Error(), ctx.Message.Reference())
		return err
	}

	profile, err := ctx.DB.GetProfile(targetUser.ID)
	if err != nil {
		return err
	}

	categories := make(map[string][]models.Task)
	var categoryOrder []string

	for _, task := range ctx.State.Tasks {
		if _, ok := categories[task.Category]; !ok {
			categoryOrder = append(categoryOrder, task.Category)
		}
		categories[task.Category] = append(categories[task.Category], task)
	}

	if len(categoryOrder) == 0 {
		_, err := ctx.Session.ChannelMessageSendReply(ctx.Message.ChannelID, "❌ Hiç görev tanımlanmamış.", ctx.Message.Reference())
		return err
	}

	firstCategory := categoryOrder[0]
	embed := createTasksEmbed(ctx, profile, firstCategory, categories[firstCategory], targetUser)
	components := createCategoryButtons("tasks", categoryOrder, firstCategory, ctx.Message.Author.ID, targetUser.ID)

	_, err = ctx.Session.ChannelMessageSendComplex(ctx.Message.ChannelID, &discordgo.MessageSend{
		Embed:      embed,
		Components: components,
		Reference:  ctx.Message.Reference(),
	})

	return err
}

func createTasksEmbed(ctx *commands.Context, profile *models.UserProfile, category string, tasks []models.Task, targetUser *discordgo.User) *discordgo.MessageEmbed {
	var builder strings.Builder

	if category == "daily" {
		builder.WriteString("> 🕒 *Günlük görevler her gece 00:00'da sıfırlanmaktadır. Günde 1 kez yapılabilir.*\n\n")
	}

	now := time.Now()
	for _, task := range tasks {
		status := "⭕"
		progress := 0

		found := false
		for _, p := range profile.Tasks {
			if p.TaskID == task.ID {
				found = true

				isCompleted := false
				if task.Category == "daily" {
					if p.Completed && p.LastCompleted.Year() == now.Year() && p.LastCompleted.YearDay() == now.YearDay() {
						isCompleted = true
					}
				} else {
					if p.Completed {
						isCompleted = true
					}
				}

				if isCompleted {
					status = "✅"
					progress = task.Requirements.TargetValue
				} else {
					msgCount := profile.MessageCount
					shrCount := profile.ShareCount
					if task.Category == "daily" {
						msgCount = profile.DailyMessageCount
						shrCount = profile.DailyShareCount
					}

					var currentStat int
					switch task.Requirements.Type {
					case "message_count":
						currentStat = msgCount
					case "share_count":
						currentStat = shrCount
					}

					baseValue := 0
					if task.Category == "daily" {
						if p.LastCompleted.Year() == now.Year() && p.LastCompleted.YearDay() == now.YearDay() {
							baseValue = p.CurrentValue
						}
					} else {
						baseValue = p.CurrentValue
					}

					progress = currentStat - baseValue
					if progress < 0 {
						progress = 0
					}
				}
				break
			}
		}

		if !found {
			msgCount := profile.MessageCount
			shrCount := profile.ShareCount
			if task.Category == "daily" {
				msgCount = profile.DailyMessageCount
				shrCount = profile.DailyShareCount
			}

			switch task.Requirements.Type {
			case "message_count":
				progress = msgCount
			case "share_count":
				progress = shrCount
			}
		}

		if progress >= task.Requirements.TargetValue {
			progress = task.Requirements.TargetValue
		}

		builder.WriteString(fmt.Sprintf("%s **%s**\n", status, task.Name))
		builder.WriteString(fmt.Sprintf("└ *%s*\n", task.Description))
		builder.WriteString(fmt.Sprintf("└ %s `%d/%d`\n", commands.GenerateProgressBar(progress, task.Requirements.TargetValue), progress, task.Requirements.TargetValue))

		var rewards []string
		for _, r := range task.Rewards {
			switch r.Type {
			case "badge":
				badge := ctx.State.GetBadge(r.Value)
				rewards = append(rewards, fmt.Sprintf("`rozet:%s`", strings.ToLower(badge.Name)))
			case "temp_access":
				rewards = append(rewards, fmt.Sprintf("`geçici_erişim:%s (%s)`", r.Value, r.Duration))
			default:
				rewards = append(rewards, fmt.Sprintf("`%s:%s`", r.Type, r.Value))
			}
		}

		builder.WriteString(fmt.Sprintf("└ Ödüller: %s\n\n", strings.Join(rewards, ", ")))
	}

	embed := &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("📝 Görevler - %s", translateCategory(category)),
		Description: builder.String(),
		Color:       0x4CAF50,
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Görevleri tamamlayarak ödüller kazanabilirsin!",
		},
	}

	if targetUser.ID != ctx.Message.Author.ID {
		displayName := targetUser.Username
		if targetUser.GlobalName != "" {
			displayName = fmt.Sprintf("%s (%s)", targetUser.GlobalName, targetUser.Username)
		}
		embed.Author = &discordgo.MessageEmbedAuthor{
			Name:    displayName,
			IconURL: targetUser.AvatarURL("128"),
		}
	}

	return embed
}
