# Add your own mini-game

<div class="file-tree" role="img" aria-label="Tug of war is three files side by side in pages/root: tug_of_war.templ, markup and Go; tug_of_war.css, its styles; tug_of_war_test.go, behaviour tests.">
<div class="file-tree-folder">📁 pages/root/</div>
<div class="file-tree-file"><span class="file-tree-name">🧩 tug_of_war.templ</span><span class="file-tree-note">markup and Go</span></div>
<div class="file-tree-file"><span class="file-tree-name">🎨 tug_of_war.css</span><span class="file-tree-note">its styles</span></div>
<div class="file-tree-file"><span class="file-tree-name">🧪 tug_of_war_test.go</span><span class="file-tree-note">behaviour tests</span></div>
</div>

Most games need no JavaScript. The snow globe shows when it is worth it.

Notes:

- Cohesion: what changes together stays together.
- JavaScript only when the phone must react on its own: sensors, dragging, physics. Animation alone is CSS.
