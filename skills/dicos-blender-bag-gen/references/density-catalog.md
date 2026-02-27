# CT Density Catalog

Material-to-density mapping and item geometry specs for all bag contents and tray items.

## Density Scale (uint16 voxel values)

| Value | Material Class | CT Appearance |
|-------|---------------|---------------|
| 0 | Air | Black/transparent |
| 800 | Very thin fabric (underwear, socks, tissues) | Barely visible |
| 1200 | Light fabric (t-shirts, chinos, shoe uppers) | Faint blue |
| 1500 | Paper/cardboard (books, magazines, business cards) | Dim |
| 1800 | Denim, rubber, leather, elastic | Slightly brighter |
| 2000 | Organic food (fruit, sandwiches, snack bars) | Visible green |
| 3000 | Bag shell (nylon/polyester/canvas) | Clear outline |
| 3500 | Liquids (water, shampoo, cosmetics, deodorant) | Teal/cyan |
| 4000 | Tray (polypropylene) | Strong outline |
| 5000 | Foil/thin aluminum | Bright |
| 7000 | Electronics (circuit boards, screens, batteries) | Amber/orange |
| 15000 | Structural metal (zippers, buckles, coins, keys, watch) | Bright white |
| 20000 | Dense/threat metal (knife blade, gun barrel) | Maximum |

## Material Name Map

```python
DENSITY_MAP = {
    'CT_BagShell': 3000, 'CT_Tray': 4000,
    'CT_Fabric_Low': 1200, 'CT_Denim': 1800,
    'CT_Underwear': 800, 'CT_Paper': 1500,
    'CT_Food': 2000, 'CT_Foil': 5000,
    'CT_Liquid_Medium': 3500, 'CT_Leather': 1800,
    'CT_Electronics': 7000, 'CT_Glass': 3500,
    'CT_Metal_High': 15000, 'CT_Threat_Metal': 20000,
}
```

## Item Catalog

### Clothing

| Item | Object Name | Geometry | Material | Notes |
|------|------------|----------|----------|-------|
| Jeans (folded) | CT_Jeans_Folded | Cube 0.14x0.13x0.008 | CT_Denim | Add rivets (cyl r=0.003) + zipper (cyl r=0.0015) as CT_Metal_High |
| Chinos (folded) | CT_Chinos_Folded | Cube 0.13x0.12x0.006 | CT_Fabric_Low | |
| T-shirt (rolled) | CT_TShirt_Rolled_N | Cylinder r=0.025 d=0.22 horiz | CT_Fabric_Low | |
| T-shirt (folded) | CT_TShirt_Folded_N | Cube 0.12x0.10x0.005 | CT_Fabric_Low | |
| Button-down shirt | CT_ButtonDown | Cube 0.12x0.11x0.005 | CT_Fabric_Low | Add buttons (cyl r=0.003) as CT_Liquid_Medium |
| Dress shirt | CT_DressShirt | Cube 0.12x0.11x0.006 | CT_Fabric_Low | Collar stays as tiny CT_Metal_High rects |
| Hoodie/sweater | CT_Hoodie | Cube 0.13x0.13x0.015 | CT_Fabric_Low | Add zipper + aglets as CT_Metal_High |
| Underwear | CT_Underwear_N | Cube 0.05x0.04x0.003 | CT_Underwear | Stack 3-5, elastic waistband torus |
| Socks (rolled pair) | CT_SockPair_N | UV sphere r=0.018 squash=(1,1,0.7) | CT_Underwear | |
| Swimsuit | CT_Swimsuit | Cube 0.08x0.06x0.003 | CT_Fabric_Low | |
| Scarf/wrap | CT_Scarf | Cylinder r=0.04 d=0.08 (rolled) | CT_Fabric_Low | |
| Belt | CT_Belt | Torus major=0.05 minor=0.005 | CT_Leather | Add CT_BeltBuckle cube 0.03x0.02x0.005 as CT_Metal_High |

### Electronics

| Item | Object Name | Geometry | Material | Notes |
|------|------------|----------|----------|-------|
| Laptop | CT_Laptop | Cube 0.005x0.14x0.09 | CT_Electronics | Add battery cube as CT_Metal_High |
| Tablet/iPad | CT_Tablet | Cube 0.004x0.09x0.065 | CT_Electronics | Add battery as CT_Metal_High |
| Phone | CT_Phone | Cube 0.003x0.035x0.07 | CT_Electronics | Add battery as CT_Metal_High |
| Power bank | CT_PowerBank | Cube 0.035x0.02x0.008 | CT_Electronics | Add cell cylinder as CT_Metal_High |
| Earbuds case | CT_EarbudsCase | UV sphere r=0.015 squash | CT_Electronics | Add battery as CT_Metal_High |
| USB cable (coiled) | CT_USBCable | Torus major=0.03 minor=0.002 | CT_Metal_High | |
| Power adapter | CT_PowerAdapter | Cube 0.02x0.015x0.01 | CT_Electronics | |
| Camera body | CT_Camera | Cube 0.06x0.04x0.035 | CT_Electronics | Dense internals |
| Camera lens | CT_CameraLens | Cylinder r=0.025 d=0.04 | CT_Glass | Glass elements |
| E-reader | CT_EReader | Cube 0.004x0.08x0.06 | CT_Electronics | Thin battery |
| Portable speaker | CT_Speaker | Cylinder r=0.03 d=0.08 | CT_Electronics | |
| Mouse | CT_Mouse | UV sphere r=0.025 squash=(1.5,1,0.6) | CT_Electronics | |
| Laptop charger | CT_LaptopCharger | Cube 0.04x0.04x0.015 | CT_Electronics | Heavy brick |

### Toiletries & Cosmetics

| Item | Object Name | Geometry | Material | Notes |
|------|------------|----------|----------|-------|
| Toiletry bag | CT_ToiletryBag | Cube 0.08x0.04x0.04 | CT_Fabric_Low | |
| Shampoo bottle | CT_Shampoo | Cylinder r=0.012 d=0.06 | CT_Liquid_Medium | |
| Toothpaste | CT_Toothpaste | Cylinder r=0.008 d=0.05 horiz | CT_Liquid_Medium | Metal cap |
| Deodorant | CT_Deodorant | Cylinder r=0.015 d=0.05 | CT_Liquid_Medium | |
| Perfume/cologne | CT_Perfume | Cube 0.012x0.012x0.02 | CT_Glass | Glass bottle |
| Lipstick | CT_Lipstick | Cylinder r=0.008 d=0.04 | CT_Metal_High | Metal tube |
| Compact mirror | CT_Compact | Cylinder r=0.025 d=0.008 | CT_Electronics | Mirror + hinge |
| Razor | CT_Razor | Cube 0.04x0.02x0.01 | CT_Liquid_Medium | Metal blade as CT_Metal_High |
| Hair brush | CT_Hairbrush | Cube 0.04x0.03x0.02 | CT_Fabric_Low | |
| Sunscreen | CT_Sunscreen | Cylinder r=0.02 d=0.06 | CT_Liquid_Medium | |
| Hand sanitizer | CT_Sanitizer | Cylinder r=0.015 d=0.04 | CT_Liquid_Medium | |
| Medication bottle | CT_MedBottle | Cylinder r=0.012 d=0.04 | CT_Liquid_Medium | |

### Food & Drink

| Item | Object Name | Geometry | Material | Notes |
|------|------------|----------|----------|-------|
| Water bottle | CT_WaterBottle | Cylinder r=0.03 d=0.18 | CT_Liquid_Medium | Metal cap CT_Metal_High |
| Sandwich | CT_Sandwich | Cube 0.06x0.04x0.025 | CT_Food | |
| Snack bars | CT_SnackBar_N | Cube 0.04x0.015x0.008 | CT_Food | Foil wrapper CT_Foil |
| Apple/fruit | CT_Apple | UV sphere r=0.035 | CT_Food | |
| Trail mix | CT_TrailMix | Cube 0.04x0.03x0.03 | CT_Food | |
| Gum/mints | CT_Gum | Cube 0.03x0.02x0.008 | CT_Food | |
| Candy bag | CT_Candy | Cube 0.05x0.03x0.02 | CT_Food | |
| Thermos | CT_Thermos | Cylinder r=0.035 d=0.20 | CT_Metal_High | Double-wall steel |
| Baby formula | CT_Formula | Cylinder r=0.03 d=0.08 | CT_Liquid_Medium | |
| Baby food jar | CT_BabyFood | Cylinder r=0.025 d=0.04 | CT_Glass | Glass jar |

### Shoes

| Item | Object Name | Geometry | Material | Notes |
|------|------------|----------|----------|-------|
| Shoe upper | CT_Shoe_N | Cube 0.06x0.045x0.025 | CT_Fabric_Low | |
| Shoe sole | CT_Sole_N | Cube 0.065x0.048x0.008 | CT_Liquid_Medium | Rubber |
| Shoe eyelets | CT_Eyelet_N | UV sphere r=0.002 | CT_Metal_High | 4-6 per shoe |
| Boot | CT_Boot_N | Cube 0.08x0.05x0.04 | CT_Leather | Taller, thicker sole |
| Sandal | CT_Sandal_N | Cube 0.06x0.04x0.008 | CT_Liquid_Medium | Mostly sole |

### Bag Hardware (Carry-On Roller)

| Item | Object Name | Geometry | Material |
|------|------------|----------|----------|
| Handle frame L/R | CT_HandleFrame_L/R | Cylinder r=0.008 d=0.20 | CT_Metal_High |
| Handle brace | CT_HandleBrace | Cylinder r=0.006 d=0.12 horiz | CT_Metal_High |
| Internal frame | CT_InternalFrame | Cube 0.25x0.002x0.10 | CT_Metal_High |
| Wheels | CT_Wheel_N | UV sphere r=0.012 | CT_Metal_High |
| Top zipper | CT_Zipper_Top | Cylinder r=0.002, bag length | CT_Metal_High |
| Side zipper | CT_Zipper_Side | Cylinder r=0.002, pocket | CT_Metal_High |
| Zipper pulls | CT_ZipperPull_N | UV sphere r=0.004 | CT_Metal_High |

### Loose Items / Misc

| Item | Object Name | Geometry | Material | Notes |
|------|------------|----------|----------|-------|
| Keys on ring | CT_Keyring + CT_Key_N | Torus + cubes | CT_Metal_High | |
| Coins | CT_Coin_N | Cylinder r=0.01 d=0.001 | CT_Metal_High | Random rotation |
| Book | CT_Book | Cube 0.06x0.04x0.01 | CT_Paper | |
| Notebook | CT_Notebook | Cube 0.07x0.05x0.008 | CT_Paper | Wire spiral as CT_Metal_High |
| Pen | CT_Pen | Cylinder r=0.004 d=0.06 | CT_Metal_High | |
| Wallet | CT_Wallet | Cube 0.015x0.055x0.045 | CT_Leather | Credit cards as thin CT_Metal_High |
| Passport | CT_Passport | Cube 0.06x0.04x0.005 | CT_Paper | RFID chip as CT_Electronics |
| Umbrella (folded) | CT_Umbrella | Cylinder r=0.02 d=0.15 | CT_Fabric_Low | Metal shaft CT_Metal_High |
| Toy (child) | CT_Toy | UV sphere r=0.03 | CT_Liquid_Medium | Plastic |
| Diaper pack | CT_Diapers | Cube 0.08x0.06x0.04 | CT_Fabric_Low | |

### Loose Tray Items (not inside bag)

| Item | Object Name | Geometry | Material | Notes |
|------|------------|----------|----------|-------|
| Watch (metal) | Tray_Watch | Torus major=0.02 minor=0.004 | CT_Metal_High | Flat on tray floor |
| Watch (leather) | Tray_Watch | Torus (band CT_Leather) + cylinder (face CT_Electronics) | Mixed | |
| Phone | Tray_Phone | Cube 0.003x0.035x0.075 | CT_Electronics | Flat, add battery |
| Laptop (separate) | Tray_Laptop | Cube 0.01x0.17x0.12 | CT_Electronics | TSA requires separate bin |
| Tablet (separate) | Tray_Tablet | Cube 0.006x0.12x0.085 | CT_Electronics | |
| Belt | Tray_Belt | Coiled torus | CT_Leather | Buckle CT_Metal_High |
| Shoes (pair) | Tray_Shoe_N | See shoes | Mixed | Side by side on tray floor |
| Wallet | Tray_Wallet | Cube 0.015x0.055x0.045 | CT_Leather | |
| Sunglasses | Tray_Sunglasses | Curved thin cube | CT_Glass | Hinge CT_Metal_High |
| Baseball cap | Tray_Cap | Hemisphere r=0.07 | CT_Fabric_Low | Metal button CT_Metal_High |
| Jacket (folded) | Tray_Jacket | Cube 0.15x0.12x0.04 | CT_Fabric_Low | Metal zippers |

### Threat Items (Optional)

| Item | Object Name | Geometry | Material | Notes |
|------|------------|----------|----------|-------|
| Knife blade | CT_Knife_Blade | Cube 0.10x0.018x0.001 | CT_Threat_Metal | Angled diagonally |
| Knife tip | CT_Knife_Tip | Cone r1=0.018 r2=0 d=0.03 flat | CT_Threat_Metal | |
| Knife handle | CT_Knife_Handle | Cube 0.05x0.012x0.006 | CT_Electronics | Composite |
| Knife tang | CT_Knife_Tang | Cube 0.05x0.008x0.002 | CT_Threat_Metal | |
| Knife rivets | CT_Knife_Rivet_N | Cylinder r=0.003 d=0.013 | CT_Threat_Metal | 3 per handle |
| Box cutter | CT_BoxCutter | Cube 0.04x0.01x0.005 | CT_Threat_Metal | Small blade |
| Scissors | CT_Scissors | 2 crossed blades | CT_Threat_Metal | |
| Multi-tool | CT_MultiTool | Cube 0.04x0.012x0.008 | CT_Threat_Metal | Very dense |
| Firearm | CT_Firearm | L-shaped cube composite | CT_Threat_Metal | Highly distinctive |

### Appliances

| Item | Object Name | Geometry | Material | Notes |
|------|------------|----------|----------|-------|
| Hair dryer barrel | CT_HairDryer_Barrel | Cylinder r=0.025 d=0.14 | CT_Fabric_Low | Plastic |
| Hair dryer motor | CT_HairDryer_Motor | Cylinder r=0.015 d=0.04 | CT_Metal_High | Dense |
| Hair dryer coils | CT_HairDryer_Coil_N | Torus major=0.018 minor=0.002 | CT_Metal_High | |
| Hair dryer handle | CT_HairDryer_Handle | Cylinder r=0.015 d=0.07 | CT_Fabric_Low | |
| Hair dryer cord | CT_HairDryer_Cord | Torus major=0.02 minor=0.003 | CT_Metal_High | |
| Curling iron | CT_CurlingIron | Cylinder r=0.012 d=0.20 | CT_Metal_High | Metal barrel |
| Electric shaver | CT_Shaver | Cube 0.03x0.025x0.06 | CT_Electronics | Motor inside |
