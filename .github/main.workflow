workflow "Pull Request" {
  on = "pull_request"
  resolves = ["GitHub Action for Slack"]
}

action "GitHub Action for Slack" {
  uses = "Ilshidur/action-slack@5faabb4216b20af98fe77b6d9048d24becfefd31"
  secrets = ["GITHUB_TOKEN"]
  env = {
    SLACK_CHANNEL = "pre-notifications"
  }
}

workflow "Pull Request Review" {
  on = "pull_request_review"
  resolves = ["GitHub Action for Slack-1"]
}

action "GitHub Action for Slack-1" {
  uses = "Ilshidur/action-slack@5faabb4216b20af98fe77b6d9048d24becfefd31"
  env = {
    SLACK_CHANNEL = "pre-notifications"
  }
}
