# Hotfix Migration 029 — Supabase Auth Trigger Ownership

Error berikut:

```text
ERROR: 42501: must be owner of table users
```

terjadi karena migration lama mencoba menjalankan `ALTER TRIGGER` pada `auth.users`.
Tabel tersebut dikelola oleh Supabase Auth dan bukan milik role SQL Editor yang
menjalankan migration.

Versi migration 029 yang telah diperbaiki hanya melakukan rebranding object pada
schema `public`. Trigger lama pada `auth.users` dibiarkan dengan nama lama. Hal ini
aman karena nama trigger bukan bagian dari API aplikasi, sedangkan hubungan trigger
dengan function PostgreSQL disimpan berdasarkan OID.

Karena migration dibungkus `BEGIN` dan `COMMIT`, kegagalan sebelumnya membatalkan
seluruh perubahan dalam migration tersebut. Jalankan ulang seluruh isi file:

```text
migrations/029_livira_rebrand_dashboard_scope_container_target.sql
```
