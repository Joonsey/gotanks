package game

import (
	"bytes"
	"encoding/json"
	"image"
	"io"
	"log"
	"math"
	"os"
	"strconv"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

const (
	STEPS = 1
)

type AssetManager struct {
	stacked_map map[image.Rectangle]string

	cached_sprites map[string][]*ebiten.Image
	rotated_cache  map[string][]*ebiten.Image

	// TODO refactor
	new_level_font *text.GoTextFaceSource
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

func (a *AssetManager) CacheRotatedSprites(path string, step int) {
	if step <= 0 {
		step = 1 // Prevent invalid step
	}

	source := a.GetSprites(path) // Retrieve sprite stack
	if _, exists := a.rotated_cache[path]; exists {
		return // Already cached
	}

	// Compute effective dimensions of the stack
	spriteWidth := source[0].Bounds().Dx()
	spriteHeight := source[0].Bounds().Dy()
	stackHeight := spriteHeight + (len(source)-1)*2                   // Account for effective height
	totalHeight := stackHeight + int(math.Ceil(float64(len(source)))) // Padding for safe bounds

	// Compute diagonal for expanded canvas
	diagonal := int(math.Ceil(math.Sqrt(float64(spriteWidth*spriteWidth + totalHeight*totalHeight))))

	// Pre-compute cache
	rotations := 360 / step
	a.rotated_cache[path] = make([]*ebiten.Image, rotations)

	for i := 0; i < rotations; i++ {
		angle := float64(i*step) * math.Pi / 180

		// Create an expanded canvas
		rotatedImage := ebiten.NewImage(diagonal, diagonal)

		for j, img := range source {
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(-float64(spriteWidth)/2, -float64(spriteHeight)/2) // Center rotation
			op.GeoM.Rotate(angle - 90*math.Pi/180)
			op.GeoM.Translate(float64(diagonal)/2, float64(diagonal)/2-float64(j)) // Center in expanded canvas
			rotatedImage.DrawImage(img, op)
		}

		a.rotated_cache[path][i] = rotatedImage
	}
}

func (a *AssetManager) DrawRotatedSprite(screen *ebiten.Image, path string, x, y, rotation float64, r, g, b, opacity float32) {
	cache := a.rotated_cache[path]
	if cache == nil || len(cache) == 0 {
		return // Cache not available
	}

	// Find the closest pre-rendered rotation
	rotations := len(cache)
	index := int(math.Round(rotation*float64(rotations)/(2*math.Pi))) % rotations
	if index < 0 {
		index += rotations
	}

	cachedSprite := cache[index]
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(x-float64(cachedSprite.Bounds().Dx())/2, y-float64(cachedSprite.Bounds().Dy())/2)
	scale := ebiten.ColorScale{}
	scale.SetR(r)
	scale.SetG(g)
	scale.SetB(b)
	op.ColorScale.ScaleAlpha(opacity)
	op.ColorScale.ScaleWithColorScale(scale)

	screen.DrawImage(cachedSprite, op)
}

func (a *AssetManager) LoadFont(path string) *text.GoTextFaceSource {
	b, err := os.ReadFile(path)
	if err != nil {
		log.Fatal(err)
	}

	s, err := text.NewGoTextFaceSource(bytes.NewReader(b))
	if err != nil {
		log.Fatal(err)
	}

	return s

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

	a.stacked_map = make(map[image.Rectangle]string)
	a.cached_sprites = make(map[string][]*ebiten.Image)
	a.rotated_cache = make(map[string][]*ebiten.Image)

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

		a.stacked_map[image.Rectangle{
			image.Point{
				x * SPRITE_SIZE, y * SPRITE_SIZE},
			image.Point{
				(1 + x) * SPRITE_SIZE, (1 + y) * SPRITE_SIZE},
		}] = value.(string)

	}

	for _, path := range a.stacked_map {
		a.CacheRotatedSprites(path, STEPS)
	}

	log.Println("succesfully loaded all stacked sprites")

	// TODO improve
	a.new_level_font = a.LoadFont("assets/fonts/PressStart2P-Regular.ttf")
}
