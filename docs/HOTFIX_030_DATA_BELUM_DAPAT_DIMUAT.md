# Hotfix 030 — “Data belum dapat dimuat” setelah rebranding

## Penyebab

Migration 029 mengganti nama object function dari prefix `sentra_` menjadi
`livira_`. PostgreSQL mempertahankan OID dan dependency object ketika function
di-rename, tetapi pemanggilan function lain yang tertulis sebagai teks di dalam
body function SQL/PLpgSQL tidak selalu ikut ditulis ulang.

Dampak utamanya:

- `livira_dashboard_summary()` masih dapat memanggil
  `public.sentra_inventory_dashboard_metrics(...)`;
- function target lama sudah berubah nama menjadi
  `public.livira_inventory_dashboard_metrics(...)`;
- PostgREST mengembalikan error dan aplikasi menampilkan pesan generik
  “Data belum dapat dimuat. Periksa konfigurasi database dan coba kembali.”

Function trigger pencarian inventory juga berpotensi masih memanggil prefix lama
ketika barang dibuat atau diperbarui.

## Perbaikan database operasional

Jalankan seluruh isi file berikut di Supabase SQL Editor:

`migrations/030_fix_rebrand_function_body_and_postgrest_cache.sql`

Setelah berhasil, restart/redeploy service Render atau lakukan manual deploy
terbaru. Migration ini aman dijalankan berulang kali dan tidak menghapus data.

## Pemeriksaan Render

Pada log request yang gagal, versi sebelum hotfix biasanya memuat salah satu
pesan berikut:

- `function public.sentra_inventory_dashboard_metrics(...) does not exist`;
- `PGRST202` / function RPC belum ditemukan pada schema cache;
- `rpc/livira_dashboard_summary` berstatus 400/404/500.

Hotfix juga mengirim `NOTIFY pgrst, 'reload schema'` agar cache RPC Supabase
diperbarui.
