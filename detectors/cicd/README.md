# CI/CD Resource Detectors

## Gitlab

Sample code snippet to initialize Gitlab resource detector

```
// Instantiate a new Gitlab CICD Resource detector
gitlabResourceDetector := gitlab.NewResourceDetector()
resource, err := gitlabResourceDetector.Detect(context.Background())
```

Gitlab CI/CD resource detector captures following Gitlab Job environment attributes

```
cicd.pipeline.name
cicd.pipeline.task.run.id
cicd.pipeline.task.name
cicd.pipeline.task.type
cicd.pipeline.run.id
cicd.pipeline.task.run.url.full
vcs.repository.ref.name
vcs.repository.ref.type
vcs.repository.change.id
vcs.repository.url.full
```
