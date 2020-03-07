# Check modified files

Retrieve modified files between 2 commits in github actions for use in later step inputs. 

## Usage:

```yaml
my-job:
  strategy:
    matrix:
      prefix:
      - pkg
      - src
  steps:
  - uses: actions-go/modified@master
    id: is-modified
    with:
      pattern: ${{ matrix.prefix }}/**/*.go
  - run: echo "${{ steps.is-modified.outputs.modified }} ${{ steps.is-modified.outputs.modified-files }}"
    if: steps.is-modified.outputs.modified == 'true'
```

## Inputs

### head

The commit to be compared to base.
This parameter is required when running on an event different from push or pullrequest.
When handling a pullrequest event, it defaults to the pullrequest head sha.
When handling a push event, it defauts to the `After` field of the push event:
  https://developer.github.com/v3/activity/events/types/#pushevent

### base

The commit head is compared to.
This parameter is required when running on an event different from push or pullrequest.
When handling a pullrequest event, it defaults to the pullrequest base sha.
When handling a push event, it defauts to the `Before` field of the push event:
  https://developer.github.com/v3/activity/events/types/#pushevent

### pattern

The pattern to which modified paths are matched.

### use-glob

Whether to use the simplee glob syntax, extended with the `**` pattern matching paths with path separator

## Outputs

### modified

true when any modified file between base and head matches pattern

### modified-files

a json encoded list of all files modified between base and head
