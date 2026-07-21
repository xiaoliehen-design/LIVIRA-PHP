# Revisi UI ukuran peti kemas

- Input ukuran peti kemas dan nomor kontainer kini memakai komponen form yang konsisten dengan kotak input lain.
- Input perkiraan volume LCL memakai gaya, radius, fokus, dan jarak yang sama.
- Opsi ukuran peti kemas: `20'`, `40'`, `40' HC`, dan `45' HC`.
- Kode lama `45` dinormalisasi menjadi `45HC` melalui migration 011.
- Perhitungan YOR: 20' = 1 TEU, 40'/40' HC = 2 TEU, dan 45' HC = 2,25 TEU.
