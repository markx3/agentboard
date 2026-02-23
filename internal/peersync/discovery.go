package peersync

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const serverInfoPath = ".agentboard/server.json"

type ServerInfo struct {
	Addr string `json:"addr"`
}

func WriteServerInfo(addr string) error {
	dir := filepath.Dir(serverInfoPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	info := ServerInfo{Addr: addr}
	data, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return err
	}

	// Write atomically via temp file
	tmp := serverInfoPath + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, serverInfoPath)
}

func ReadServerInfo() (*ServerInfo, error) {
	data, err := os.ReadFile(serverInfoPath)
	if err != nil {
		return nil, fmt.Errorf("no server info: %w", err)
	}

	var info ServerInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, fmt.Errorf("invalid server info: %w", err)
	}
	return &info, nil
}

func RemoveServerInfo() error {
	return os.Remove(serverInfoPath)
}
