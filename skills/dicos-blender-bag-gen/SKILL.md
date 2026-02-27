---
name: dicos-blender-bag-gen
description: >
  Generate 3D CT scan visualizations of bags and personal items in airport
  screening trays using Blender MCP. Supports varied container types (carry-on
  suitcases, backpacks, purses, laptop bags, duffel bags) and loose tray items
  (watches, phones, laptops, belts, shoes). Creates realistic randomized packing
  with CT density-based materials, and optionally voxelizes to raw volume data
  for DICOS export. Use when the user asks to: create a bag scan, generate a CT
  bag, build a screening scene, make a bag in Blender, simulate an airport
  X-ray/CT scan, add items to a bag, voxelize a Blender scene, or generate
  screening training data. Requires Blender MCP connection.
---

# CT Bag Generator

Generate realistic CT scan visualizations of bags and personal items in TSA screening trays via Blender MCP.

## Prerequisites

Requires [Blender MCP](https://github.com/ahujasid/blender-mcp) - a Model Context Protocol server that connects Claude to Blender. See [references/blender-mcp-setup.md](references/blender-mcp-setup.md) for full install instructions (download addon.py, install uv, configure `~/.claude.json`, activate in Blender).

All scene creation uses `mcp__blender__execute_blender_code` and `mcp__blender__get_viewport_screenshot`.

## Workflow

### 1. Choose a Scenario

Pick or randomize a traveler profile. Each produces different bag types and contents.

| Profile | Container | Typical Contents |
|---------|-----------|-----------------|
| Business trip | Carry-on roller + laptop bag | Suits, dress shirts, laptop, tablet, chargers, toiletries |
| Vacation | Large carry-on roller | Casual clothes, swimwear, sunscreen, snacks, camera |
| Weekend getaway | Backpack or duffel | 1-2 changes, toiletries, phone charger, book |
| Student | Backpack | Laptop, textbooks, cables, snacks, water bottle |
| Parent w/ child | Duffel or tote | Snacks, toys, diapers, wipes, change of clothes, tablet |
| Commuter | Messenger/laptop bag | Laptop, notebook, lunch, water bottle, keys, wallet |

A second tray for loose personal items is common: watch, belt, phone, wallet, shoes.

### 2. Create Screening Tray

See [references/tray-specs.md](references/tray-specs.md) for exact TSA tray dimensions.

**Key rule: nothing penetrates the tray.** The tray is the outermost container. All items (bags and loose objects) sit ON the tray floor, inside the tray walls, with clearance gaps.

```
Tray interior floor = tray_z_min + wall_thickness (~4mm)
Container bottom Z  = tray floor + 1mm clearance
Container XY extent < tray interior XY (with ~10mm gap per side)
```

### 3. Create Container

See [references/container-types.md](references/container-types.md) for geometry and hardware per container type.

Containers are **hollow shells** in the CT volume (2-voxel thick). The shell density (3000) makes the bag outline visible; interior is filled with contents or air.

### 4. Pack Contents with Randomness

See [references/density-catalog.md](references/density-catalog.md) for the full item catalog.

**Randomization rules** - simulate how real people pack:
- Vary item count: use `random.randint(min, max)` for each category
- Offset positions: add `random.uniform(-0.02, 0.02)` jitter to X/Y placement
- Rotate items: add `random.uniform(-15, 15)` degree tilt to folded clothes
- Skip categories: not every bag has shoes, food, or a hoodie
- Mix folding styles: some items rolled, some folded flat, some crumpled (scaled unevenly)
- Overlap slightly: real packed clothes compress against each other
- Stuff gaps: socks and underwear get tucked into corners and shoe cavities

**Packing is bottom-up.** Track a running Z height as items stack. Heavier/flatter items at the bottom, bulky/light items on top. Electronics go in designated pockets (back of bag for laptops, front pocket for tablets/phones).

### 5. Add Loose Tray Items (Optional)

Items that go directly in the tray, NOT inside a bag:

| Item | Geometry | Density | Notes |
|------|----------|---------|-------|
| Watch (metal band) | Torus major=0.02 minor=0.004 | 15000 | Flat on tray floor |
| Watch (leather band) | Torus major=0.02 minor=0.004 | 1800 band / 7000 face | |
| Phone | Cube 0.003x0.035x0.075 | 7000 + 15000 battery | Flat on tray |
| Laptop (removed) | Cube 0.01x0.17x0.12 | 7000 + 15000 battery | TSA requires separate |
| Tablet | Cube 0.006x0.12x0.085 | 7000 + 15000 battery | |
| Belt | Cylinder r=0.008, coiled torus | 1800 leather / 15000 buckle | |
| Shoes (removed) | See catalog | 1200 + 1800 sole | Pair side by side |
| Wallet | Cube 0.015x0.055x0.045 | 1800 leather / 15000 cards | |
| Sunglasses | Curved cube | 3500 lens / 15000 hinge | |
| Baseball cap | Hemisphere r=0.07 | 1200 | With 15000 metal button |

Place loose items with random rotation on the tray floor. They must not overlap each other or the bag.

### 6. Constrain All Items

After placing everything:
1. Clamp bag contents inside bag bounds (8mm inset margin)
2. Clamp loose tray items inside tray bounds (8mm inset margin)
3. Verify NO object penetrates the tray shell from below
4. Verify bag shell does not overlap tray shell

### 7. Apply CT Materials

See density-catalog.md for the full material table. Principled BSDF + emission per tier. Bag shell and tray are **opaque** (alpha=1.0).

### 8. Add Threat Items (Optional)

See [references/threat-items.md](references/threat-items.md) for the full TSA prohibited items catalog with Blender geometry and CT signatures.

Threat items use DICOS Threat Categories per NEMA IIC 1 v04-2023: `METAL` (firearms, knives), `EXPLOSIVE` (IEDs, detonators), `CONTRABAND` (tools, weapons, sporting goods), `ANOMALY` (suspicious configurations).

When placing threats, record their world-space bounding boxes for the TDR (step 10).

### 9. Voxelize (Optional)

See [references/voxelization.md](references/voxelization.md).

Critical voxelization rule: **tray mask is built first**, then bag and all other objects exclude tray voxels. This prevents interleaving.

Also export a `threats.json` sidecar with bounding box coordinates for any threat items placed in step 8.

### 10. Export to DICOS (Optional)

See [references/dicos-export.md](references/dicos-export.md) for the complete conversion workflow with Go code examples.

Uses [github.com/jpfielding/dicos.go](https://github.com/jpfielding/dicos.go) to write:
- **CT Image** (`.dcs`): Multi-frame volume, one axial slice per frame, with ImagePlane geometry, scanner parameters, and HU rescale
- **TDR** (`.dcs`): Threat Detection Report with PTO bounding boxes referencing the CT Image, if threats are present

A bundled converter is included at [scripts/voxel2dicos/](scripts/voxel2dicos/). Copy it to the project and run:
```bash
go run ./scripts/voxel2dicos/ tmp/voxels.raw tmp/bag_ct.dcs
```

The script reads the raw binary header + voxel data, builds a DICOS CTImage with proper modules (Patient, Study, Series, Equipment, ImagePlane, FrameOfReference, CTImage, VOILUT), and writes it as an uncompressed multi-frame `.dcs` file. See the Go source for the full tag mapping.

### 11. Generate TDR with Threat Boxes (Optional)

See [references/tdr-workflow.md](references/tdr-workflow.md) for DICOS TDR specifics.

A TDR is a separate DICOS file that references the CT image and marks threat regions with 3D bounding boxes. Each Potential Threat Object (PTO) includes a Threat Category (NEMA IIC 1 v04-2023 Defined Terms), Assessment Flag (`HIGH_THREAT`/`THREAT`), probability, and bounding box coordinates in mm.

## Naming Convention

- Tray: `ScreeningTray`
- Container: `CarryOnBag`, `Backpack`, `Purse`, `DuffelBag`, `LaptopBag`, `MessengerBag`
- Bag contents: `CT_` prefix (e.g., `CT_Laptop`, `CT_Jeans_Folded`)
- Loose tray items: `Tray_` prefix (e.g., `Tray_Watch`, `Tray_Phone`, `Tray_Belt`)
- Hardware sub-parts: parent name + suffix (e.g., `CT_LaptopBattery`, `CT_HoodieZipper`)
