# TSA Screening Tray Specifications

## Standard Tray Sizes

Trays vary by manufacturer and checkpoint. These are the common types:

| Type | Outer (L x W x H) | Interior (L x W x H) | Wall Thick | Use |
|------|-------------------|----------------------|------------|-----|
| Standard flat | 660 x 420 x 75mm | 640 x 400 x 65mm | ~10mm | Loose items, small bags |
| Deep bin | 640 x 420 x 150mm | 620 x 400 x 140mm | ~10mm | Carry-on bags, shoes |
| Large deep bin | 700 x 480 x 150mm | 680 x 460 x 140mm | ~10mm | Oversize bags |

## Scanner Tunnel Constraints

Smiths HI-SCAN 6040 CTiX (common checkpoint scanner):
- Tunnel opening: 620mm W x 420mm H
- Belt width: 600mm
- Tray must fit through tunnel, so max tray width ~700mm

## Blender Geometry

All dimensions in meters (Blender default units). Place tray at origin.

```python
# Deep bin tray (most common for bags)
TRAY_OUTER = (0.640, 0.420, 0.150)  # L x W x H in meters * scale
TRAY_WALL = 0.010   # 10mm walls
TRAY_FLOOR = 0.004  # 4mm floor thickness

# In Blender scale (1 unit = 1 meter):
# Outer half-extents for primitive_cube_add
tray_hx = 0.320   # ±320mm
tray_hy = 0.210   # ±210mm
tray_hz = 0.075   # 75mm half-height, bottom at Z=0, top at Z=0.150
```

Build the tray as a box with a boolean-subtracted inner cavity, or as a shell mesh.

## Non-Penetration Rules

1. **Tray is the absolute boundary.** Nothing extends below Z=0 or outside tray outer walls.
2. **Tray floor clearance.** Items sit on the interior floor at `Z = TRAY_FLOOR + 0.001` (1mm gap).
3. **Wall clearance.** Items must be at least 5-10mm inside tray interior walls.
4. **No mesh overlap.** During voxelization, tray mask is authoritative - tray voxels are never overwritten.

## Tray Material

- Injection-molded polypropylene
- CT density: 4000 (clearly visible outline)
- Color in Blender: opaque dark blue-gray
- Typically gray or dark gray in reality
