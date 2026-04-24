---
name: getting-started
description: How to get started with Tharsis? This guide covers all the details for getting started with tharsis
---

The user is new to Tharsis and you need to walk them through the following quickstart guide. Do not execute any tools. The goal is to have the user execute these commands as you guide them through it.

## Apply a sample Terraform module

1) Download the tharsis cli from: https://gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-cli/-/releases

2) Next create a workspace. In order to create a workspace you'll need access to a root level group

3) Create a terraform configuration which will be deployed using tharsis. To create the terraform configuration do the following:

Create a new directory and save the following as `module.tf`:

```hcl showLineNumbers title="Sample Terraform Module using null resource"
# Simulate creating a resource which takes a minute.
resource "time_sleep" "wait_60_seconds" {
  create_duration = "60s"
}

resource "null_resource" "next" {
  depends_on = [time_sleep.wait_60_seconds]
}
```

```shell title="Apply the Terraform module"
tharsis apply -directory-path "/path/to/directory/containing/module/file" <parent-group>/<subgroup>/<workspace>
```

🔥🔥 Congratulations! You've just learned the basics of Tharsis 🔥🔥

For more information checkout the youtube video here https://www.youtube.com/watch?v=zhkfyRugk_I or the tharsis docs at https://tharsis.martian-cloud.io/
