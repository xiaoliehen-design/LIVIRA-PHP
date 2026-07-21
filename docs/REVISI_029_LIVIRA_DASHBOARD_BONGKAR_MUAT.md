# Revisi 029 — LIVIRA, Cakupan Dashboard, dan Target Bongkar/Muat

## Rebranding LIVIRA

Seluruh identitas yang terlihat pengguna telah diubah menjadi:

**LIVIRA — Layanan Inventori, Verifikasi, Integrasi, Rekonsiliasi, dan Analitik**

Perubahan mencakup halaman login, sidebar, judul browser, konfigurasi default, nama file ekspor, metadata Excel, dokumentasi, nama folder proyek, RPC database baru, serta bucket dokumen baru `livira-documents`.

## Cakupan inventory dashboard

Dropdown **Cakupan inventory** sekarang berurutan sebagai berikut:

1. **Masih di TPS** — hanya barang aktif yang belum berada di TPP.
2. **Seluruh TPP** — seluruh barang aktif yang sudah berada di TPP.
3. **TPP Transporindo**, **TPP Multi Sejahtera**, dan seterusnya — hanya barang aktif pada TPP terpilih.

Setiap perubahan pilihan menghitung ulang **Total inventory aktif, BTD, BDN, BMMN, Barang Titipan**, serta rincian dokumen/FCL/LCL dari dataset yang sama. Bug pada role administrator yang sebelumnya melewati filter dashboard telah diperbaiki.

## Bongkar/muat per kontainer

- Kontainer FCL hanya tampil satu kali pada daftar target meskipun memiliki beberapa uraian barang.
- Setelah kontainer dipilih, seluruh uraian di dalam kontainer muncul pada panel alokasi.
- Setiap uraian wajib memiliki tujuan dan total kuantitas tujuan harus sama dengan kuantitas sumber.
- Backend menolak request parsial yang mencoba memproses hanya sebagian uraian dalam kontainer.
- Barang LCL tetap dipilih dan diproses per uraian barang.
- Tampilan daftar, mode dropdown, scrollbar, kartu target, dan panel alokasi dibuat lebih halus dan konsisten.

## Cara implementasi database

### Database operasional yang sudah pernah dipakai

Jalankan:

```text
migrations/029_livira_rebrand_dashboard_scope_container_target.sql
```

Migration 029 aman dijalankan ulang. Migration ini mengganti nama function/trigger/index teknis pada schema `public` ke prefix `livira_`, memastikan RPC bongkar/muat tersedia, dan membuat bucket privat `livira-documents`. Trigger lama pada `auth.users` sengaja dipertahankan karena tabel tersebut dimiliki layanan Auth Supabase dan nama trigger tidak memengaruhi proses aplikasi. Dokumen lama tetap dapat diunduh karena metadata setiap dokumen menyimpan nama bucket asal.

Setelah itu ubah environment Render:

```text
APP_NAME=LIVIRA
SUPABASE_STORAGE_BUCKET=livira-documents
```

### Database Supabase baru dan kosong

Jalankan satu file berikut saja:

```text
migrations/01_SETUP_DATABASE_BARU_KOSONG_LIVIRA_001_032.sql
```
