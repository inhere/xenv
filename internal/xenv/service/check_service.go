package service

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/inhere/xenv/internal/xenv/models"
	"github.com/inhere/xenv/internal/xenv/sdk"
)

type CheckStatus string

const (
	CheckStatusOK    CheckStatus = "ok"
	CheckStatusWarn  CheckStatus = "warn"
	CheckStatusError CheckStatus = "error"
)

type ToolRequirement struct {
	MinVersion string
	Required   bool
}

type CheckResult struct {
	Name    string
	Status  CheckStatus
	Message string
}

type CheckService struct {
	sdks *SDKService
}

func NewCheckService(sdks *SDKService) *CheckService {
	return &CheckService{sdks: sdks}
}

func ParseToolRequirement(raw string) (ToolRequirement, error) {
	req := ToolRequirement{Required: true}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return req, fmt.Errorf("empty tool requirement")
	}
	if raw == "*" {
		return req, nil
	}

	parts := strings.Split(raw, ",")
	for i, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		if i == 0 {
			switch {
			case part == "*":
				req.MinVersion = ""
			case strings.HasPrefix(part, ">="):
				req.MinVersion = strings.TrimSpace(strings.TrimPrefix(part, ">="))
				if req.MinVersion == "" {
					return req, fmt.Errorf("invalid tool requirement: %q", raw)
				}
			default:
				return req, fmt.Errorf("unsupported tool requirement: %q", raw)
			}
			continue
		}

		switch strings.ToLower(part) {
		case "required":
			req.Required = true
		case "optional":
			req.Required = false
		default:
			return req, fmt.Errorf("unsupported tool requirement option: %q", part)
		}
	}

	return req, nil
}

func (s *CheckService) CheckTools(state *models.ActivityState, checkVersion bool) []CheckResult {
	results := make([]CheckResult, 0, len(state.ToolRequirements))
	for name, raw := range state.ToolRequirements {
		req, err := ParseToolRequirement(raw)
		if err != nil {
			results = append(results, CheckResult{
				Name:    name,
				Status:  CheckStatusWarn,
				Message: err.Error(),
			})
			continue
		}

		path, err := exec.LookPath(name)
		if err != nil {
			status := CheckStatusWarn
			if req.Required {
				status = CheckStatusError
			}
			results = append(results, CheckResult{
				Name:    name,
				Status:  status,
				Message: "not found in PATH",
			})
			continue
		}

		msg := "found at " + path
		if req.MinVersion != "" && checkVersion {
			verOut, verErr := exec.Command(name, "--version").CombinedOutput()
			if verErr != nil {
				results = append(results, CheckResult{
					Name:    name,
					Status:  CheckStatusWarn,
					Message: "failed to read version: " + strings.TrimSpace(string(verOut)),
				})
				continue
			}

			current := extractVersionString(string(verOut))
			if current == "" {
				results = append(results, CheckResult{
					Name:    name,
					Status:  CheckStatusWarn,
					Message: "version output could not be parsed",
				})
				continue
			}
			if sdk.CompareVersions(current, req.MinVersion) < 0 {
				status := CheckStatusWarn
				if req.Required {
					status = CheckStatusError
				}
				results = append(results, CheckResult{
					Name:    name,
					Status:  status,
					Message: fmt.Sprintf("version %s is lower than required %s", current, req.MinVersion),
				})
				continue
			}
			msg = fmt.Sprintf("%s, version %s", msg, current)
		} else if req.MinVersion != "" {
			msg = fmt.Sprintf("%s, version requirement >=%s not checked", msg, req.MinVersion)
		}

		results = append(results, CheckResult{
			Name:    name,
			Status:  CheckStatusOK,
			Message: msg,
		})
	}
	return results
}

func (s *CheckService) CheckSDKs(state *models.ActivityState) []CheckResult {
	results := make([]CheckResult, 0, len(state.SDKs))
	if s == nil || s.sdks == nil {
		return results
	}

	for name, version := range state.SDKs {
		_, err := s.sdks.WhereSDK(name+":"+version, true)
		if err != nil {
			results = append(results, CheckResult{
				Name:    name,
				Status:  CheckStatusError,
				Message: err.Error(),
			})
			continue
		}
		results = append(results, CheckResult{
			Name:    name,
			Status:  CheckStatusOK,
			Message: "sdk is available",
		})
	}
	return results
}

func extractVersionString(out string) string {
	fields := strings.Fields(out)
	for _, field := range fields {
		field = strings.Trim(field, "vV(),")
		if field == "" {
			continue
		}
		hasDigit := false
		valid := true
		for _, r := range field {
			if r >= '0' && r <= '9' {
				hasDigit = true
				continue
			}
			if r != '.' {
				valid = false
				break
			}
		}
		if hasDigit && valid {
			return field
		}
	}
	return ""
}
