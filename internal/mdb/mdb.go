package mdb

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

var base64StdToRe = strings.NewReplacer(
	"+", "!",
	"/", "-",
)

var base64ReToStd = strings.NewReplacer(
	"!", "+",
	"-", "/",
)

type MachineDB struct {
	mu       sync.RWMutex
	basePath string
}

func NewMachineDB(storePath string) (*MachineDB, error) {
	stat, err := os.Stat(storePath)
	if err != nil && errors.Is(err, fs.ErrNotExist) {
		err = os.MkdirAll(storePath, 0700)
	}

	switch {
	case err == nil && !stat.IsDir():
		err = errors.New("is not a directory")
		fallthrough
	case err != nil:
		return nil, fmt.Errorf("invalid store path: %w", err)
	}

	return &MachineDB{
		basePath: storePath,
	}, nil
}

func (m *MachineDB) List() ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	files, err := os.ReadDir(m.basePath)
	if err != nil {
		return nil, fmt.Errorf("unable to read store dir: %w", err)
	}

	out := make([]string, 0, len(files))
	for _, file := range files {
		machineID := file.Name()
		if !strings.HasPrefix(machineID, "m_") {
			continue
		}

		machineID = strings.TrimPrefix(machineID, "m_")
		machineID = strings.TrimSuffix(machineID, ".json")
		out = append(out, base64ReToStd.Replace(machineID))
	}

	return out, nil
}

func (m *MachineDB) IsMachineExists(machineFP string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	info, err := os.Stat(m.storePath(machineFP))
	return err == nil && !info.IsDir()
}

func (m *MachineDB) Get(machineFP string, keyID string) ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	machineData, err := m.getAllLocked(machineFP)
	if err != nil {
		return nil, err
	}

	out, ok := machineData[keyID]
	if !ok {
		return nil, fmt.Errorf("key %q for machine %q was not found", keyID, machineFP)
	}

	return out, nil
}

func (m *MachineDB) Put(machineFP string, keyID string, data []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	machineData, err := m.getAllLocked(machineFP)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}

	if machineData == nil {
		machineData = make(map[string][]byte)
	}

	machineData[keyID] = data
	rawData, err := json.Marshal(machineData)
	if err != nil {
		return fmt.Errorf("unable to marshal michine data: %w", err)
	}

	return os.WriteFile(m.storePath(machineFP), rawData, 0600)
}

func (m *MachineDB) storePath(machineFP string) string {
	filename := fmt.Sprintf("m_%s.json", base64StdToRe.Replace(machineFP))
	return filepath.Join(m.basePath, filename)
}

func (m *MachineDB) getAllLocked(machineFP string) (map[string][]byte, error) {
	rawData, err := os.ReadFile(m.storePath(machineFP))
	if err != nil {
		return nil, fmt.Errorf("unable to get machine file: %w", err)
	}

	var out map[string][]byte
	if err := json.Unmarshal(rawData, &out); err != nil {
		return nil, fmt.Errorf("invalid machine data: %w", err)
	}

	return out, nil
}
