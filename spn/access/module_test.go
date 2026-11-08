package access

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/safing/portmaster/base/config"
	"github.com/safing/portmaster/base/database"
	"github.com/safing/portmaster/base/database/storage/hashmap"
	"github.com/safing/portmaster/service/mgr"
	"github.com/safing/portmaster/spn/conf"
)

type testInstance struct {
	config   *config.Config
	spnGroup *mgr.ExtendedGroup
}

func (stub *testInstance) Config() *config.Config             { return stub.config }
func (stub *testInstance) SPNGroup() *mgr.ExtendedGroup       { return stub.spnGroup }
func (stub *testInstance) Stopping() bool                     { return false }
func (stub *testInstance) IsShuttingDown() bool               { return false }
func (stub *testInstance) SetCmdLineOperation(f func() error) {}
func (stub *testInstance) DataDir() string                    { return _dataDir }

var _dataDir string

func TestMain(m *testing.M) {
	exitCode := 1
	defer func() {
		if exitCode != 0 {
			os.Exit(exitCode)
		}
	}()

	var err error
	// Create a temporary directory for the data
	_dataDir, err = os.MkdirTemp("", "")
	if err != nil {
		fmt.Printf("failed to create temporary data directory: %s", err)
		return // Exit with error
	}
	defer func() { _ = os.RemoveAll(_dataDir) }()

	// Inject a memory database so these unit tests do not apply production
	// filesystem ACLs to storage subdirectories on Windows.
	err = database.Initialize(_dataDir)
	if err != nil {
		fmt.Printf("initialize database: %s", err)
		return
	}
	_, err = database.Register(&database.Database{
		Name:        "core",
		Description: "Holds core data, such as settings and profiles",
		StorageType: database.StorageTypeInjected,
	})
	if err != nil {
		fmt.Printf("failed to register core database: %s", err)
		return // Exit with error
	}
	storage, err := hashmap.NewHashMap("core", "")
	if err != nil {
		fmt.Printf("create memory database: %s", err)
		return
	}
	_, err = database.InjectDatabase("core", storage)
	if err != nil {
		fmt.Printf("inject memory database: %s", err)
		return
	}

	// Initialize the instance
	instance := &testInstance{spnGroup: mgr.NewExtendedGroup()}
	conf.EnableIntegration(true)

	instance.config, err = config.New(instance)
	if err != nil {
		fmt.Printf("failed to create config module: %s", err)
		return // Exit with error
	}
	// Simulate a configuration loaded from an older SPN-enabled installation.
	err = config.Register(&config.Option{Name: "Legacy SPN", Description: "Legacy SPN configuration regression fixture.", Key: "spn/enable", OptType: config.OptTypeBool, DefaultValue: true})
	if err != nil {
		fmt.Printf("register legacy setting: %s", err)
		return
	}
	module, err = New(instance)
	if err != nil {
		fmt.Printf("failed to create access module: %s", err)
		return // Exit with error
	}

	err = instance.config.Start()
	if err != nil {
		fmt.Printf("failed to start config module: %s", err)
		return // Exit with error
	}
	err = module.Start()
	if err != nil {
		fmt.Printf("failed to start access module: %s", err)
		return // Exit with error
	}

	// Let the asynchronous local feature publication finish before tests touch
	// the shared account cache. This must not block the service startup path.
	deadline := time.Now().Add(5 * time.Second)
	for {
		_, err = GetUser()
		if err == nil {
			break
		}
		if time.Now().After(deadline) {
			_, lastErr := GetUser()
			fmt.Printf("local feature plan was not published: %v\n", lastErr)
			return
		}
		time.Sleep(time.Millisecond)
	}
	exitCode = m.Run()
}
