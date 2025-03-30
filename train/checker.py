# from PIL import Image

# img = Image.open("alien.train.tif")

# lines = open("alien.train.box").readlines()

# for line in lines:
#     l = line.split()
#     left = int(l[1])
#     bottom = int(l[2])
#     right = int(l[3])
#     top = int(l[4])
#     cropped = img.crop((left, 4800-top, right, 4800-bottom))
#     cropped.save(f"{l[0]}.png")