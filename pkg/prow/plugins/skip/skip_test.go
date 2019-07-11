/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package skip

import (
	"reflect"
	"testing"

	"github.com/sirupsen/logrus"

	"github.com/jenkins-x/lighthouse/pkg/prow/config"
	"github.com/jenkins-x/lighthouse/pkg/prow/fakegithub"
	"github.com/jenkins-x/lighthouse/pkg/prow/github"
)

func TestSkipStatus(t *testing.T) {
	tests := []struct {
		name string

		presubmits     []config.Presubmit
		sha            string
		event          *github.GenericCommentEvent
		prChanges      map[int][]*scm.Change
		existing       []*scm.Status
		combinedStatus string
		expected       []*scm.Status
	}{
		{
			name: "required contexts should not be skipped regardless of their state",

			presubmits: []config.Presubmit{
				{
					Reporter: config.Reporter{
						Context: "passing-tests",
					},
				},
				{
					Reporter: config.Reporter{
						Context: "failed-tests",
					},
				},
				{
					Reporter: config.Reporter{
						Context: "pending-tests",
					},
				},
			},
			sha: "shalala",
			event: &github.GenericCommentEvent{
				IsPR:       true,
				IssueState: "open",
				Action:     scm.ActionCreate,
				Body:       "/skip",
				Number:     1,
				Repo:       scm.Repository{Namespace: "org", Name: "repo"},
			},
			existing: []*scm.Status{
				{
					Context: "passing-tests",
					State:   scm.StateSuccess,
				},
				{
					Context: "failed-tests",
					State:   scm.StatusFailure,
				},
				{
					Context: "pending-tests",
					State:   scm.StatusPending,
				},
			},

			expected: []*scm.Status{
				{
					Context: "passing-tests",
					State:   scm.StateSuccess,
				},
				{
					Context: "failed-tests",
					State:   scm.StatusFailure,
				},
				{
					Context: "pending-tests",
					State:   scm.StatusPending,
				},
			},
		},
		{
			name: "optional contexts that have failed or are pending should be skipped",

			presubmits: []config.Presubmit{
				{
					Optional: true,
					Reporter: config.Reporter{
						Context: "failed-tests",
					},
				},
				{
					Optional: true,
					Reporter: config.Reporter{
						Context: "pending-tests",
					},
				},
			},
			sha: "shalala",
			event: &github.GenericCommentEvent{
				IsPR:       true,
				IssueState: "open",
				Action:     scm.ActionCreate,
				Body:       "/skip",
				Number:     1,
				Repo:       scm.Repository{Namespace: "org", Name: "repo"},
			},
			existing: []*scm.Status{
				{
					State:   scm.StatusFailure,
					Context: "failed-tests",
				},
				{
					State:   scm.StatusPending,
					Context: "pending-tests",
				},
			},

			expected: []*scm.Status{
				{
					State:       scm.StateSuccess,
					Description: "Skipped",
					Context:     "failed-tests",
				},
				{
					State:       scm.StateSuccess,
					Description: "Skipped",
					Context:     "pending-tests",
				},
			},
		},
		{
			name: "optional contexts that have not posted a context should not be skipped",

			presubmits: []config.Presubmit{
				{
					Optional: true,
					Reporter: config.Reporter{
						Context: "untriggered-tests",
					},
				},
			},
			sha: "shalala",
			event: &github.GenericCommentEvent{
				IsPR:       true,
				IssueState: "open",
				Action:     scm.ActionCreate,
				Body:       "/skip",
				Number:     1,
				Repo:       scm.Repository{Namespace: "org", Name: "repo"},
			},
			existing: []*scm.Status{},

			expected: []*scm.Status{},
		},
		{
			name: "optional contexts that have succeeded should not be skipped",

			presubmits: []config.Presubmit{
				{
					Optional: true,
					Reporter: config.Reporter{
						Context: "succeeded-tests",
					},
				},
			},
			sha: "shalala",
			event: &github.GenericCommentEvent{
				IsPR:       true,
				IssueState: "open",
				Action:     scm.ActionCreate,
				Body:       "/skip",
				Number:     1,
				Repo:       scm.Repository{Namespace: "org", Name: "repo"},
			},
			existing: []*scm.Status{
				{
					State:   scm.StateSuccess,
					Context: "succeeded-tests",
				},
			},

			expected: []*scm.Status{
				{
					State:   scm.StateSuccess,
					Context: "succeeded-tests",
				},
			},
		},
		{
			name: "optional tests that have failed but will be handled by trigger should not be skipped",

			presubmits: []config.Presubmit{
				{
					Optional:     true,
					Trigger:      `(?m)^/test (?:.*? )?job(?: .*?)?$`,
					RerunCommand: "/test job",
					Reporter: config.Reporter{
						Context: "failed-tests",
					},
				},
			},
			sha: "shalala",
			event: &github.GenericCommentEvent{
				IsPR:       true,
				IssueState: "open",
				Action:     scm.ActionCreate,
				Body: `/skip
/test job`,
				Number: 1,
				Repo:   scm.Repository{Namespace: "org", Name: "repo"},
			},
			existing: []*scm.Status{
				{
					State:   scm.StatusFailure,
					Context: "failed-tests",
				},
			},
			expected: []*scm.Status{
				{
					State:   scm.StatusFailure,
					Context: "failed-tests",
				},
			},
		},
		{
			name: "no contexts should be skipped if the combined status is success",

			presubmits: []config.Presubmit{
				{
					Optional: true,
					Reporter: config.Reporter{
						Context: "failed-tests",
					},
				},
				{
					Optional: true,
					Reporter: config.Reporter{
						Context: "pending-tests",
					},
				},
			},
			sha:            "shalala",
			combinedStatus: scm.StateSuccess,
			event: &github.GenericCommentEvent{
				IsPR:       true,
				IssueState: "open",
				Action:     scm.ActionCreate,
				Body:       "/skip",
				Number:     1,
				Repo:       scm.Repository{Namespace: "org", Name: "repo"},
			},
			existing: []*scm.Status{
				{
					State:   scm.StatusFailure,
					Context: "failed-tests",
				},
				{
					State:   scm.StatusPending,
					Context: "pending-tests",
				},
			},
			expected: []*scm.Status{
				{
					State:   scm.StatusFailure,
					Context: "failed-tests",
				},
				{
					State:   scm.StatusPending,
					Context: "pending-tests",
				},
			},
		},
	}
	for _, test := range tests {
		if err := config.SetPresubmitRegexes(test.presubmits); err != nil {
			t.Fatalf("%s: could not set presubmit regexes: %v", test.name, err)
		}

		fghc := &fakegithub.FakeClient{
			IssueComments: make(map[int][]*scm.Comment),
			PullRequests: map[int]*scm.PullRequest{
				test.event.Number: {
					Head: scm.PullRequestBranch{
						Sha: test.sha,
					},
				},
			},
			PullRequestChanges: test.prChanges,
			CreatedStatuses: map[string][]*scm.Status{
				test.sha: test.existing,
			},
			CombinedStatuses: map[string]*scm.CombinedStatus{
				test.sha: {
					State:    test.combinedStatus,
					Statuses: test.existing,
				},
			},
		}
		l := logrus.WithField("plugin", pluginName)

		if err := handle(fghc, l, test.event, test.presubmits, true); err != nil {
			t.Errorf("%s: unexpected error: %v", test.name, err)
			continue
		}

		// Check that the correct statuses have been updated.
		created := fghc.CreatedStatuses[test.sha]
		if len(test.expected) != len(created) {
			t.Errorf("%s: status mismatch: expected:\n%+v\ngot:\n%+v", test.name, test.expected, created)
			continue
		}
		for _, got := range created {
			var found bool
			for _, exp := range test.expected {
				if exp.Context == got.Context {
					found = true
					if !reflect.DeepEqual(exp, got) {
						t.Errorf("%s: expected status: %v, got: %v", test.name, exp, got)
					}
				}
			}
			if !found {
				t.Errorf("%s: expected context %q in the results: %v", test.name, got.Context, created)
				break
			}
		}
	}
}