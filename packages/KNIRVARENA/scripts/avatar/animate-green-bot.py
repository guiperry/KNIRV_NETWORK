"""Add a fitted robot skeleton and looping clips, preserving original mesh/materials.
Usage: python3 scripts/avatar/animate-green-bot.py [original.glb] [output.glb]
Requires numpy. Input must be the original, unrigged explorer asset.
"""
import json
import math
from pathlib import Path
import struct
import sys
import numpy as np

DEFAULT = Path(__file__).resolve().parents[2] / 'public/assets/avatar/Green_Bot_Explorer.glb'
source = Path(sys.argv[1]) if len(sys.argv) > 1 else DEFAULT
output = Path(sys.argv[2]) if len(sys.argv) > 2 else DEFAULT
raw = source.read_bytes()
length = struct.unpack_from('<I', raw, 12)[0]
gltf = json.loads(raw[20:20 + length])
if gltf.get('skins'):
    raise SystemExit('Already rigged; provide the original asset to regenerate.')
data = bytearray(raw[28 + length:])

def accessor(values, kind, component=5126):
    array = np.asarray(values, dtype='<f4' if component == 5126 else '<u2')
    while len(data) % 4:
        data.append(0)
    view = len(gltf['bufferViews'])
    gltf['bufferViews'].append({'buffer': 0, 'byteOffset': len(data), 'byteLength': array.nbytes})
    data.extend(array.tobytes())
    entry = {'bufferView': view, 'componentType': component, 'count': len(array), 'type': kind}
    if kind == 'SCALAR':
        entry.update(min=[float(array.min())], max=[float(array.max())])
    gltf['accessors'].append(entry)
    return len(gltf['accessors']) - 1

primitive = gltf['meshes'][0]['primitives'][0]
a = gltf['accessors'][primitive['attributes']['POSITION']]
v = gltf['bufferViews'][a['bufferView']]
points = np.frombuffer(data, dtype='<f4', count=a['count'] * 3,
                       offset=v.get('byteOffset', 0) + a.get('byteOffset', 0)).reshape(-1, 3).copy()
# Pivots measured from the source mesh. The robot faces +Z, Y is up.
bones = [('Body', None, (0, 0, 0)), ('Head', 0, (0, .33, 0))]
for side, x in [('L', .46), ('R', -.46)]:
    shoulder = len(bones)
    bones.extend([(f'Arm{side}', 0, (x, .235, 0)),
                  (f'Forearm{side}', shoulder, (x, -.055, 0))])
    hip = len(bones)
    bones.extend([(f'Thigh{side}', 0, (x * .43, -.37, 0)),
                  (f'Shin{side}', hip, (x * .43, -.64, 0))])
base = len(gltf['nodes'])
for name, parent, pivot in bones:
    origin = np.array(bones[parent][2]) if parent is not None else np.zeros(3)
    gltf['nodes'].append({'name': name, 'translation': (np.array(pivot) - origin).tolist()})
    if parent is not None:
        gltf['nodes'][base + parent].setdefault('children', []).append(len(gltf['nodes']) - 1)
gltf['scenes'][gltf.get('scene', 0)]['nodes'].append(base)
# Blend narrowly at mechanical joints; keep shells rigid elsewhere.
joints = np.zeros((len(points), 4), dtype=np.uint16)
weights = np.zeros((len(points), 4), dtype=np.float32)
weights[:, 0] = 1
x, y, z = points.T

def region(mask, child, parent, pivot, width=.045):
    blend = np.clip((pivot + width / 2 - y[mask]) / width, 0, 1)
    joints[mask, 0] = child
    joints[mask, 1] = parent
    weights[mask, 0] = blend
    weights[mask, 1] = 1 - blend

head = y > .30
joints[head, 0] = 1
for positive, arm, thigh in [(True, 2, 4), (False, 6, 8)]:
    side = (x > 0) if positive else (x < 0)
    arm_mask = side & (np.abs(x) > .355) & (y < .30)
    joints[arm_mask, 0] = arm
    region(arm_mask & (y < -.025), arm + 1, arm, -.055)
    leg_mask = side & (y < -.365) & ((np.abs(x) > .12) | (y < -.53))
    region(leg_mask, thigh, 0, -.38)
    region(leg_mask & (y < -.61), thigh + 1, thigh, -.64)
primitive['attributes']['JOINTS_0'] = accessor(joints, 'VEC4', 5123)
primitive['attributes']['WEIGHTS_0'] = accessor(weights, 'VEC4')
inverse = []
for _, _, pivot in bones:
    m = np.eye(4); m[:3, 3] = -np.array(pivot)
    inverse.append(m.T.reshape(16))
gltf['skins'] = [{'name': 'ExplorerRig', 'joints': list(range(base, base + len(bones))),
                  'skeleton': base, 'inverseBindMatrices': accessor(inverse, 'MAT4')}]
gltf['nodes'][0]['skin'] = 0
gltf['nodes'][0]['name'] = 'ExplorerMesh'
gltf['animations'] = []

def quaternion(rx=0, ry=0, rz=0):
    cx, cy, cz = [math.cos(a / 2) for a in (rx, ry, rz)]
    sx, sy, sz = [math.sin(a / 2) for a in (rx, ry, rz)]
    return [sx*cy*cz+cx*sy*sz, cx*sy*cz-sx*cy*sz, cx*cy*sz+sx*sy*cz, cx*cy*cz-sx*sy*sz]

for name, duration in [('Idle', 2.4), ('Run', .64), ('Work', 1.2)]:
    times = np.linspace(0, duration, 49)
    time_index = accessor(times, 'SCALAR')
    animation = {'name': name, 'channels': [], 'samplers': []}
    def track(bone, path, values, kind):
        sampler = len(animation['samplers'])
        animation['samplers'].append({'input': time_index, 'output': accessor(values, kind), 'interpolation': 'LINEAR'})
        animation['channels'].append({'sampler': sampler, 'target': {'node': base + bone, 'path': path}})
    for bone, (label, _, _) in enumerate(bones):
        rotations = []
        for t in times:
            phase = t / duration * math.tau
            s = math.sin(phase)
            opposite = 1 if label.endswith('L') else -1
            rx = ry = rz = 0
            if name == 'Idle':
                if label == 'Head': ry = .07*s
                if label.startswith('Arm'): rz = opposite * .025*s
            elif name == 'Run':
                if label == 'Body': rx = .12; rz = .035*s
                if label == 'Head': rx = -.09
                if label.startswith('Thigh'): rx = opposite * .68*s
                if label.startswith('Shin'): rx = .12 + .95*max(0, -opposite*s)
                if label.startswith('Arm'): rx = -opposite*.58*s; rz = opposite*.08
                if label.startswith('Forearm'): rx = -.65 - .18*opposite*s
            else:
                if label == 'Body': rx = .10 + .025*math.sin(phase*2)
                if label == 'Head': rx = .16; ry = .09*s
                if label.startswith('Arm'): rx = -.65 + opposite*.16*s; rz = opposite*.13
                if label.startswith('Forearm'): rx = -.85 + opposite*.30*s
                if label.startswith('Thigh'): rx = -.08
                if label.startswith('Shin'): rx = .16
            rotations.append(quaternion(rx, ry, rz))
        track(bone, 'rotation', rotations, 'VEC4')
    offsets = []
    for t in times:
        phase = t / duration * math.tau
        height = (.045*(1-math.cos(phase*2)) if name == 'Run' else .008*math.sin(phase))
        offsets.append([0, height, 0])
    track(0, 'translation', offsets, 'VEC3')
    gltf['animations'].append(animation)
gltf['buffers'][0]['byteLength'] = len(data)
encoded = json.dumps(gltf, separators=(',', ':')).encode()
encoded += b' ' * (-len(encoded) % 4)
data += b'\0' * (-len(data) % 4)
output.write_bytes(struct.pack('<III', 0x46546c67, 2, 28 + len(encoded) + len(data)) +
                   struct.pack('<II', len(encoded), 0x4e4f534a) + encoded +
                   struct.pack('<II', len(data), 0x004e4942) + data)
print(f'Wrote {output}: {len(bones)} bones; Idle, Run, Work')
