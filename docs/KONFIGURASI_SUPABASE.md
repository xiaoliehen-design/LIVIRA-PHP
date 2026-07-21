# Konfigurasi Supabase untuk LIVIRA PHP

LIVIRA PHP menggunakan project Supabase yang sama melalui:

- Auth API untuk login, OTP, reset password, dan admin delete user;
- PostgREST untuk tabel/view;
- RPC untuk transaksi atomik inventory dan proses;
- Storage API untuk dokumen private.

## Nilai yang diperlukan

Dari Supabase Project Settings/API:

- Project URL → `SUPABASE_URL`
- anon/public key → `SUPABASE_ANON_KEY`
- service role key → `SUPABASE_SERVICE_ROLE_KEY`

Bucket dokumen:

```env
SUPABASE_STORAGE_BUCKET=livira-documents
```

Pastikan bucket yang digunakan sama dengan aplikasi lama. Jangan membuat bucket baru bila dokumen produksi sudah berada pada bucket lama.

## Keamanan key

- anon key boleh dipakai untuk Auth user, tetapi paket ini tetap menyimpannya di backend.
- service role key dapat melewati RLS dan wajib menjadi secret server.
- jangan commit `.env`.
- bila key pernah terunggah ke GitHub, segera rotate key Supabase.

## Database lama

Tidak perlu mengubah schema hanya karena backend menjadi PHP. View, function, trigger, RLS, data, dan RPC tetap digunakan.

Jalankan SQL setup hanya untuk project Supabase baru yang benar-benar kosong. File reset data bersifat destruktif dan tidak diperlukan untuk migrasi backend.
