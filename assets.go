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

	cached_sprites map[string][]*ebiten.Image
}

func (a *AssetManager) GetSprites(path string) []*ebiten.Image {
	sprites, ok := a.cached_sprites[path]
	if ok {
		return sprites
	}

	sprite, _, err := ebitenutil.NewImageFromFile(path)
	if err != nil {
		log.Fatal(err)
	}

	// caching requested sprites for future
	sprite_split := SplitSprites(sprite)
	a.cached_sprites[path] = sprite_split

	return sprite_split
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
	a.cached_sprites = make(map[string][]*ebiten.Image)

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

		sprite := a.GetSprites(value.(string))

		a.stacked_map[image.Rectangle{
			image.Point{
				x * SPRITE_SIZE, y * SPRITE_SIZE},
			image.Point{
				(1 + x) * SPRITE_SIZE, (1 + y) * SPRITE_SIZE},
		}] = sprite
	}

	log.Println("succesfully loaded all stacked sprites")
}
