# ![blanche](blanche.png)

**blanche** is a tool that automates the portion of your [GitOps](https://www.gitops.tech/) workflow in Kubernetes between your CI pipeline, which creates your Docker images, and your CD pipeline, which deploys your Docker images.

Take the following GitOps workflow:

**CI**

1. Open Pull Request with changes to `guestbook` application
2. Pull Request merged to `master`
3. CI Pipeline builds docker image and pushes to DockerHub as `guestbook:v2`

**CD**

1. Helm charts for deploying `guestbook` live in `cd-configs` repo. CD Pipeline will automatically deploy changes to Kubernetes cluster.
2. When ready to deploy `v2`, create new Pull Request and update `guestbook` helm chart with:

    ```diff
    image:
    - tag: v1
    + tag: v2
    ```

3. Pull Request approved and merged
4. `guestbook:v2` deployed successfully

## The Problem

This GitOps workflow is not fully-automated, it still requires manual intervention to create a change to the CD repo to get the new tag deployed via a commit or a Pull request.

## The Solution

**blanche** automates the changing of a Docker tag within a Helm chart, whenever a new Docker tag is created. Since GitOps requires that you have a declarative state of your system, any changes to applications running in your cluster should be appriopriately tagged.

**blanche** helps in getting these tags updated automatically. She can push a tag change directly to specific branches, or create a Pull Request for the change.

This means that new tags can be automatically deployed to a cluster using a CD tool, without manually changing the config files, while maintaining the "single source of truth" as required by GitOps.

**blanche** is meant for use where deploys to specific environments require approval, while other environments don't.

## About

**blanche** is **NOT** a CI or CD tool. She is an automation tool that sits between your CI and CD tools and links them together.

**blanche** does not care about what CI or CD tool you use. There are plenty out there, have your pick. All she does is take metadata about an artifact your CI creates and updates your CD configs.

**blanche** currently only supports the following situations:

* Configs are managed by Helm values files, and tags are stored in the following structure:

  ```yaml
  image:
    tag: THIS_GETS_UPDATED
  ```

* Relies on webhooks send from a Docker registry. Currently [Docker Hub](https://docs.docker.com/docker-hub/webhooks/) is the only supported registry.
* Only supports updating CD configs in GitHub.

## Requirements

* GitHub Access Token with `write` access to your CD config repo(s) (environment variable `GITHUB_ACCESS_TOKEN`).
* Ingress/External URL for DockerHub to successfully send webhooks
* Manifest definitions for which configs to updated based on which docker image is updated (see [manifest-example.yaml](manifest-example.yaml))
  * The file path for this file can be customized by envirnoment variable `MANIFEST_PATH`

### Example Flow Diagram

```
          ┌───────────────────────────┐
          │                           │
          │      guestbook.git        │
          │                           │
          └───────────────────────────┘
                        │
                        │
                   app master
                   branch is
                    updated
                        │
                        ▼
         ┌────────────────────────────┐
         │ CI pipeline builds Docker  │
         │      image with tag        │
         └────────────────────────────┘
                        │
           DockerHub sends webhook with
            new image tag to blanche
                        │
                        │
                        ▼
     ┌────────────────────────────────────┐
     │  blanche creates a new commit in   │
     │  cd-configs based on the manifest  │
     │             definitions            │
     │                                    │
     └────────────────────────────────────┘   ┌──────────────────────┐
                        │                     │  Developer directly  │
                        │                     │ modifies Helm values │
              git push or new PR   ┌──────────│ to update Kubernetes │
                      push         │          │     resource(s)      │
                        │          │          └──────────────────────┘
                        ▼          ▼
        ┌───────────────────────────────┐
        │          cd-configs.git       │
        │            updated            │
        └───────────────────────────────┘
                        │
                        │
                        ▼
        ┌───────────────────────────────┐
        │   CD Tool sees changes and    │
        │     deploys accordingly       │
        └───────────────────────────────┘
```

## Contributing

Bug reports and pull requests are welcome on GitHub at <https://github.com/RentTheRunway/blanche>.

## License

The application is available as open source under the terms of the [MIT License](LICENSE).

### Development

See [Makefile](Makefile) for running blanche in dev mode.
