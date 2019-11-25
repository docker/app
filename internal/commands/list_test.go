package commands

import (
	"testing"
	"time"

	"github.com/deislabs/cnab-go/claim"
	"github.com/docker/app/internal/store"
	"gotest.tools/assert"
)

type mockInstallationStore struct {
	installations map[string]*store.Installation
}

func (m mockInstallationStore) List() ([]string, error) {
	l := []string{}
	for k := range m.installations {
		l = append(l, k)
	}
	return l, nil
}
func (m mockInstallationStore) Store(installation *store.Installation) error { return nil }
func (m mockInstallationStore) Read(installationName string) (*store.Installation, error) {
	return m.installations[installationName], nil
}
func (m mockInstallationStore) Delete(installationName string) error { return nil }

type stubServiceFetcher struct{}

func (s stubServiceFetcher) getServices(*store.Installation) (appServices, error) {
	return map[string]ServiceStatus{"service1": {DesiredTasks: 1, RunningTasks: 1}}, nil
}

func TestGetInstallationsSorted(t *testing.T) {
	now := time.Now()
	oldInstallation := &store.Installation{
		Claim: claim.Claim{
			Name:     "old-installation",
			Modified: now.Add(-1 * time.Hour),
		},
	}
	newInstallation := &store.Installation{
		Claim: claim.Claim{
			Name:     "new-installation",
			Modified: now,
		},
	}
	installationStore := mockInstallationStore{installations: map[string]*store.Installation{"old-installation": oldInstallation, "new-installation": newInstallation}}
	installations, err := getInstallations(installationStore, &stubServiceFetcher{})
	assert.NilError(t, err)
	assert.Equal(t, len(installations), 2)
	// First installation is the last modified
	assert.Equal(t, installations[0].Name, "new-installation")
	assert.Equal(t, installations[0].Services["service1"].DesiredTasks, 1)
	assert.Equal(t, installations[1].Name, "old-installation")
	assert.Equal(t, installations[1].Services["service1"].RunningTasks, 1)
}

func TestPrintServices(t *testing.T) {
	testCases := []struct {
		name         string
		installation Installation
		expected     string
	}{
		{
			"Failed installation",
			Installation{},
			"N/A",
		},
		{
			"Non running service",
			Installation{Services: map[string]ServiceStatus{
				"service1": {DesiredTasks: 1, RunningTasks: 0},
			}},
			"0/1",
		},
		{
			"Mixed running services and non running",
			Installation{Services: map[string]ServiceStatus{
				"service1": {DesiredTasks: 1, RunningTasks: 0},
				"service2": {DesiredTasks: 5, RunningTasks: 1},
			}},
			"1/2",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			output := printServices(testCase.installation)
			assert.Equal(t, testCase.expected, output)
		})
	}
}
