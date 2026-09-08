# Explorer animations

`public/assets/avatar/Green_Bot_Explorer.glb` contains a fitted 10-bone skeleton
and three looping, in-place clips: `Idle` (2.4s), `Run` (0.64s), and `Work` (1.2s).
The original geometry, UVs, and materials are preserved. Narrow skin-weight blends
at the elbows, hips, and knees articulate the robot's mechanical parts.

`AIAgent` clones skeletons independently, blends clips over 0.18s, turns toward
its assigned node, and moves at 2.8 world units/second. It stops 1.4 units from
the node and persists its arrival position with status `working`. Assigning a
staged bot also deploys it. The anchor-straightening sequence uses the same rig.

Run `npm run dev`, then open:

- `/arena/scripts/avatar/preview.html`: orbit the model, select clips, or play an approach-and-repair demo.
- `/arena/scripts/avatar/game-preview.html`: isolated real `AIAgent` and store integration; click **Deploy to error node**.

These development previews do not connect to a production game session and are
not included in the production build.

To regenerate, provide an **unrigged** copy of the original GLB:

```sh
python3 scripts/avatar/animate-green-bot.py /tmp/original-explorer.glb public/assets/avatar/Green_Bot_Explorer.glb
```

The authoring script requires Python 3 and NumPy. It refuses already-rigged input.
No Blender or GIMP installation is needed. The resulting GLB is editable in any
3D animation tool supporting glTF skins and animation clips.
