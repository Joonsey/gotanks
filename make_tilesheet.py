from PIL import Image
import os

TILE_SIZE = 16

def sprite_stack(images: list[Image.Image]) -> Image.Image:
    """
    Stacks a list of images with a vertical offset to simulate 3D sprite stacking.

    Args:
    - images: List of Pillow Image objects representing the individual slices.
    - vertical_offset: Pixels to offset each layer in the stack (default is 4).

    Returns:
    - A new Image object with the stacked images.
    """
    # Assuming all images are the same size
    width, _ = images[0].size
    stacked_height = 32

    # Create a new image to stack on (RGBA mode for transparency support)
    stacked_image = Image.new('RGBA', (width, stacked_height))

    # Paste each image slice on top of each other with an offset
    for i, img in enumerate(images[::-1]):
        # Use img as mask for transparency
        stacked_image.paste(img, (0, -i), img)

    return stacked_image


def create_spritesheet(stacked_images: list[Image.Image], sheet_width: int = 12) -> Image.Image:
    """
    Combines a list of stacked images into a single spritesheet.

    Args:
    - stacked_images: List of stacked images to be placed on the spritesheet.
    - sheet_width: Number of images per row in the spritesheet.

    Returns:
    - A Pillow Image object representing the complete spritesheet.
    """
    # Get the dimensions of each stacked image (assuming all the same size)
    # img_width, img_height = stacked_images[0].size
    img_width, img_height = 16, 16

    # Calculate spritesheet dimensions
    rows = (len(stacked_images) + sheet_width -
            1) // sheet_width  # ceil(len/sheet_width)

    sheet_height = img_height * rows
    sheet_width_px = img_width * sheet_width

    # Create the spritesheet image
    spritesheet = Image.new('RGBA', (sheet_width_px, sheet_height))

    # Paste each stacked image into the spritesheet
    for index, img in enumerate(stacked_images):
        x = (index % sheet_width) * img_width
        y = (index // sheet_width) * img_height
        spritesheet.paste(img, (x, y), img)  # Use img as mask for transparency

    return spritesheet


def load_image(path: str) -> list[Image.Image]:
    """
    Loads an array images from file.

    Args:
    - image_paths: file path to the PNG images.

    Returns:
    - List of Pillow Image objects.
    """
    image = Image.open(path).convert('RGBA')
    images = []
    w, h = image.size
    for i in range(0, h // TILE_SIZE):
        images.append(image.crop((0, i * TILE_SIZE, w, (1 + i) * TILE_SIZE)))
    return images


def save_image(image: Image.Image, path: str) -> None:
    """
    Saves a Pillow Image object to a file.

    Args:
    - image: Pillow Image object to be saved.
    - path: Path where the image should be saved.
    """
    image.save(path, 'PNG')


# Example usage
if __name__ == '__main__':
    STACK_PATH = 'assets/sprites/stacks/'
    OUTPUT_PATH = 'assets/tiled/stacked_tilemap.png'
    image_paths = os.listdir(STACK_PATH)

    sprite_slices = [STACK_PATH + path for path in image_paths if path.endswith("png")]

    # Load the images and stack them
    stacked_images = []
    for slices in sprite_slices:
        images = load_image(slices)
        stacked_image = sprite_stack(images)  # Adjust offset as needed
        stacked_images.append(stacked_image)

    # Create a spritesheet (e.g., 4 images per row)
    spritesheet = create_spritesheet(stacked_images)

    # Save the spritesheet to a file
    save_image(spritesheet, OUTPUT_PATH)

    print("Spritesheet created and saved as '%s'" % OUTPUT_PATH)
