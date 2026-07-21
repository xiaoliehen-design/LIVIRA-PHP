# Validasi Data Workbook BCP

Tiga workbook referensi telah diperiksa sebelum model data dan workflow aplikasi dibuat.

## Ringkasan pemeriksaan

| Workbook | Baris data terisi | Hasil |
|---|---:|---|
| `BCP BTD.xlsx` | 9 | Struktur data dan action berhasil dipetakan. Formula status pada data sumber memiliki referensi rusak (`#REF!`). |
| `BCP BDN.xlsx` | 0 | Template dan seluruh kelompok action berhasil dipetakan. |
| `BCP BMMN.xlsx` | 0 | Template dan seluruh kelompok action berhasil dipetakan. |

## Normalisasi yang diterapkan

- Status tidak lagi dihitung dengan formula Excel. Setiap action membuat record `events`, dan status inventory mengikuti event terakhir.
- Pencatatan BTD/Penetapan BDN menjadi pintu masuk pertama ke inventory.
- BMMN tidak dapat dibuat langsung; `origin_type` BTD atau BDN tetap disimpan setelah penetapan BMMN.
- Lelang, pemusnahan, dan hibah/PSP wajib dimulai dari `inventory_id` aktif.
- Nomor dokumen, tanggal dokumen, catatan, petugas, dan timestamp disimpan pada timeline.

## Catatan master TPP dan data GitHub

Nama TPP pada sebagian baris workbook BTD tidak sama dengan master empat TPP aplikasi. Aplikasi hanya memakai:

1. TPP Transporindo
2. TPP Multi Sejahtera
3. TPP KBN Marunda
4. TPP Graha Segara

Data consignee, alamat, dan barang asli dari workbook tidak ditanam ke seed repository agar tidak ikut terpublikasi ketika source diunggah ke GitHub dan agar baris tidak dipetakan ke TPP yang salah. `migrations/002_seed.sql` berisi data fiktif untuk demo. Data operasional dapat dimasukkan melalui form Pencatatan BTD/Penetapan BDN setelah nama TPP sumber dinormalisasi ke master yang benar.

