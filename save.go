package game

import (
	"encoding/gob"
	"log"
	"os"
	"path/filepath"
)

const (
	SAVE_DIR_NAME  = "gotanks"
	SAVE_FILE_NAME = "save.gob"
)

type SaveManager struct {
	data SaveData

	save_file_path string
}

type SaveData struct {
	Player_ID [16]byte
}

func InitSaveManager() *SaveManager {
	sm := SaveManager{}
	save_dir := PrepareSaveDirectory(SAVE_DIR_NAME)
	save_file_path := filepath.Join(save_dir, SAVE_FILE_NAME)

	data, err := loadGameData(save_file_path)
	if err != nil {
		log.Panic(err)
	}

	sm.save_file_path = save_file_path
	sm.data = data
	return &sm
}

func (sm *SaveManager) Save() {
	err := saveGameData(sm.save_file_path, sm.data)
	if err != nil {
		log.Panic(err)
	}
}

func PrepareSaveDirectory(save_dir_name string) (save_dir_path string) {
	config_dir, err := os.UserConfigDir()
	if err != nil {
		log.Panic("Error getting config directory:", err)
		return
	}
	save_dir_path = filepath.Join(config_dir, save_dir_name)
	if _, err := os.Stat(save_dir_path); os.IsNotExist(err) {
		err := os.MkdirAll(save_dir_path, 0755)
		if err != nil {
			log.Panic("Error creating save directory:", err)
		}
	}
	return save_dir_path
}

func (sm *SaveManager) IsFresh() bool {
	return sm.data == SaveData{}
}

func saveGameData(file_path string, data SaveData) error {
	file, err := os.Create(file_path)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := gob.NewEncoder(file)
	log.Println("saved to file: ", file_path)

	err = encoder.Encode(data)
	if err != nil {
		return err
	}

	return nil
}

func loadGameData(filePath string) (SaveData, error) {
	var data SaveData

	info, err := os.Stat(filePath)
	if os.IsNotExist(err) || info.Size() == 0 {
		log.Println("Save file does not exist or is empty; returning default data.")
		return data, nil
	}

	file, err := os.Open(filePath)
	if err != nil {
		return data, err
	}
	defer file.Close()

	decoder := gob.NewDecoder(file)
	err = decoder.Decode(&data)
	if err != nil {
		return data, err
	}

	return data, nil
}
