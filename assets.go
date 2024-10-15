package main

import (
	"encoding/json"
	"image"
	"io"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

type AssetManager struct {
	stacked_map map[image.Rectangle][]*ebiten.Image
}

func (a *AssetManager) Init(config_path string) {
	jsonFile, err := os.Open(config_path)
	if err != nil {
		log.Fatalf("Failed to open JSON file: %s", err)
	}
	defer jsonFile.Close()

	// Read the JSON file
	byteValue, err := io.ReadAll(jsonFile)
	if err != nil {
		log.Fatalf("Failed to read JSON file: %s", err)
	}

	// Unmarshal the JSON data into a map[string]interface{}
	var result map[string]interface{}
	err = json.Unmarshal(byteValue, &result)
	if err != nil {
		log.Fatalf("Failed to unmarshal JSON: %s", err)
	}

	a.stacked_map = make(map[image.Rectangle][]*ebiten.Image)

	// Iterate over the map and print key-value pairs
	for key, value := range result {
		keys := strings.Split(key, ", ")
		x, err := strconv.Atoi(keys[0])
		if err != nil {
			log.Fatalf("Failed to convert to INT: %s", err)
		}

		y, err := strconv.Atoi(keys[1])
		if err != nil {
			log.Fatalf("Failed to convert to INT: %s", err)
		}

		sprite, _, err := ebitenutil.NewImageFromFile(value.(string))

		a.stacked_map[image.Rectangle{
			image.Point{
				x * SPRITE_SIZE, y * SPRITE_SIZE},
			image.Point{
				(1 + x) * SPRITE_SIZE, (1 + y) * SPRITE_SIZE},
		}] = SplitSprites(sprite)
	}

	log.Println("succesfully loaded all stacked sprites")
}
