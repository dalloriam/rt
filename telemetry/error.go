package telemetry

import "errors"

var ErrMissingTeamName = errors.New("`HONEYCOMB_TEAM_NAME` environment variable is missing")
