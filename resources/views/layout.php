<!doctype html>
<html lang="id">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <meta name="theme-color" content="#102a43">
  <meta name="description" content="LIVIRA — Layanan Inventori, Verifikasi, Integrasi, Rekonsiliasi, dan Analitik">
  <?php if (tpl_truthy(tpl_not(tpl_get($ctx, 'AuthPage')))): ?><meta name="csrf-token" content="<?= tpl_escape(tpl_get($ctx, 'CSRF')) ?>">
  <meta name="idle-timeout-seconds" content="<?= tpl_escape(tpl_get($ctx, 'IdleTimeoutSeconds')) ?>"><?php endif; ?>
  <title><?= tpl_escape(tpl_get($ctx, 'Title')) ?> · LIVIRA</title>
  <link rel="icon" href="/assets/favicon.svg" type="image/svg+xml">
  <link rel="stylesheet" href="/assets/app.css">
</head>
<body class="<?php if (tpl_truthy(tpl_get($ctx, 'AuthPage'))): ?>auth-body<?php else: ?>app-body<?php endif; ?>">
<?php if (tpl_truthy(tpl_get($ctx, 'AuthPage'))): ?>
  <?= $content ?>
<?php else: ?>
  <div class="mobile-overlay" data-sidebar-overlay></div>
  <aside class="sidebar" data-sidebar>
    <div class="brand">
      <span class="brand-mark" aria-hidden="true">
        <svg viewBox="0 0 32 32"><path d="M6 8.5 16 3l10 5.5v15L16 29 6 23.5z"/><path d="M16 3v26M6 8.5l10 5.4 10-5.4M6 23.5l10-5.3 10 5.3"/></svg>
      </span>
      <span><strong>LIVIRA</strong><small>Inventori · Verifikasi · Analitik</small></span>
    </div>

    <nav class="nav" aria-label="Menu utama">
      <p class="nav-label">Ruang kerja</p>
      <?php if (tpl_truthy(tpl_can(tpl_get($ctx, 'User'), 'dashboard.view'))): ?><a class="nav-item <?php if (tpl_truthy(tpl_eq(tpl_get($ctx, 'Active'), 'dashboard'))): ?>active<?php endif; ?>" href="/">
        <svg viewBox="0 0 24 24"><rect x="3" y="3" width="7" height="7" rx="2"/><rect x="14" y="3" width="7" height="7" rx="2"/><rect x="3" y="14" width="7" height="7" rx="2"/><rect x="14" y="14" width="7" height="7" rx="2"/></svg>
        <span>Dashboard</span>
      </a><?php endif; ?>
      <?php if (tpl_truthy(tpl_can(tpl_get($ctx, 'User'), 'inventory.view'))): ?><a class="nav-item <?php if (tpl_truthy(tpl_eq(tpl_get($ctx, 'Active'), 'inventory'))): ?>active<?php endif; ?>" href="/inventory">
        <svg viewBox="0 0 24 24"><path d="M4 7.5 12 3l8 4.5v9L12 21l-8-4.5z"/><path d="m4 7.5 8 4.5 8-4.5M12 12v9"/></svg>
        <span>Inventory</span>
      </a><?php endif; ?>
      <?php if (tpl_truthy(tpl_or(tpl_can(tpl_get($ctx, 'User'), 'auction.view'), tpl_can(tpl_get($ctx, 'User'), 'destruction.view')))): ?><p class="nav-label nav-label-spaced">Penyelesaian</p><?php endif; ?>
      <?php if (tpl_truthy(tpl_can(tpl_get($ctx, 'User'), 'auction.view'))): ?><a class="nav-item <?php if (tpl_truthy(tpl_eq(tpl_get($ctx, 'Active'), 'lelang'))): ?>active<?php endif; ?>" href="/proses/lelang">
        <svg viewBox="0 0 24 24"><path d="m14 5 5 5M12.5 6.5l5 5M4 20l7-7M9 4l11 11M3 21h7"/></svg>
        <span>Lelang</span>
      </a><?php endif; ?>
      <?php if (tpl_truthy(tpl_can(tpl_get($ctx, 'User'), 'destruction.view'))): ?><a class="nav-item <?php if (tpl_truthy(tpl_eq(tpl_get($ctx, 'Active'), 'musnah'))): ?>active<?php endif; ?>" href="/proses/musnah">
        <svg viewBox="0 0 24 24"><path d="M4 7h16M9 7V4h6v3M6 7l1 14h10l1-14M10 11v6M14 11v6"/></svg>
        <span>Musnah</span>
      </a><?php endif; ?>
      <?php if (tpl_truthy(tpl_can(tpl_get($ctx, 'User'), 'grant.view'))): ?><a class="nav-item <?php if (tpl_truthy(tpl_eq(tpl_get($ctx, 'Active'), 'hibah'))): ?>active<?php endif; ?>" href="/proses/hibah">
        <svg viewBox="0 0 24 24"><path d="M20.8 8.4c0 5.2-8.8 10.1-8.8 10.1S3.2 13.6 3.2 8.4A4.4 4.4 0 0 1 12 8a4.4 4.4 0 0 1 8.8.4Z"/><path d="M12 8v5M9.5 10.5h5"/></svg>
        <span>Hibah</span>
      </a><?php endif; ?>
      <?php if (tpl_truthy(tpl_can(tpl_get($ctx, 'User'), 'reconciliation.view'))): ?><a class="nav-item <?php if (tpl_truthy(tpl_eq(tpl_get($ctx, 'Active'), 'rekonsiliasi'))): ?>active<?php endif; ?>" href="/rekonsiliasi">
        <svg viewBox="0 0 24 24"><path d="M4 7h12M4 12h8M4 17h10M18 6v12M15 15l3 3 3-3"/></svg>
        <span>Rekonsiliasi</span>
      </a><?php endif; ?>
      <?php if (tpl_truthy(tpl_can(tpl_get($ctx, 'User'), 'reports.view'))): ?><a class="nav-item <?php if (tpl_truthy(tpl_eq(tpl_get($ctx, 'Active'), 'pelaporan'))): ?>active<?php endif; ?>" href="/pelaporan">
        <svg viewBox="0 0 24 24"><path d="M5 3h10l4 4v14H5z"/><path d="M15 3v5h5M8 12h8M8 16h8"/></svg>
        <span>Pelaporan</span>
      </a><?php endif; ?>
      <?php if (tpl_truthy(tpl_can(tpl_get($ctx, 'User'), 'search.view'))): ?><a class="nav-item <?php if (tpl_truthy(tpl_eq(tpl_get($ctx, 'Active'), 'pencarian'))): ?>active<?php endif; ?>" href="/pencarian">
        <svg viewBox="0 0 24 24"><circle cx="10.5" cy="10.5" r="6.5"/><path d="m16 16 4.5 4.5M8 8h5M8 11h3"/></svg>
        <span>Pencarian Detail Barang</span>
      </a><?php endif; ?>
      <?php if (tpl_truthy(tpl_or(tpl_can(tpl_get($ctx, 'User'), 'admin.users'), tpl_can(tpl_get($ctx, 'User'), 'admin.roles')))): ?>
      <p class="nav-label nav-label-spaced">Administrasi</p>
      <?php if (tpl_truthy(tpl_can(tpl_get($ctx, 'User'), 'admin.users'))): ?><a class="nav-item <?php if (tpl_truthy(tpl_eq(tpl_get($ctx, 'Active'), 'admin-users'))): ?>active<?php endif; ?>" href="/admin/pendaftaran"><svg viewBox="0 0 24 24"><path d="M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="m17 11 2 2 4-4"/></svg><span>Setujui Pendaftaran</span></a><?php endif; ?>
      <?php if (tpl_truthy(tpl_can(tpl_get($ctx, 'User'), 'admin.roles'))): ?><a class="nav-item <?php if (tpl_truthy(tpl_eq(tpl_get($ctx, 'Active'), 'admin-roles'))): ?>active<?php endif; ?>" href="/admin/roles"><svg viewBox="0 0 24 24"><path d="M12 3 4 7v5c0 5 3.4 8 8 9 4.6-1 8-4 8-9V7z"/><path d="M9 12h6M12 9v6"/></svg><span>Role & Hak Akses</span></a><?php endif; ?>
      <?php if (tpl_truthy(tpl_can(tpl_get($ctx, 'User'), 'admin.parameters'))): ?><a class="nav-item <?php if (tpl_truthy(tpl_eq(tpl_get($ctx, 'Active'), 'admin-parameters'))): ?>active<?php endif; ?>" href="/admin/parameters"><svg viewBox="0 0 24 24"><path d="M4 7h10M18 7h2M4 17h2M10 17h10M14 4v6M8 14v6"/></svg><span>Parameter Sistem</span></a><?php endif; ?>
      <?php endif; ?>
    </nav>

    <div class="sidebar-foot">
      <div class="mode-card">
        <span class="mode-dot"></span>
        <span><strong><?php if (tpl_truthy(tpl_get($ctx, 'DemoMode'))): ?>Mode Demo<?php else: ?>Database Aktif<?php endif; ?></strong><small><?php if (tpl_truthy(tpl_get($ctx, 'DemoMode'))): ?>Data aman untuk dicoba<?php else: ?>Tersambung ke Supabase<?php endif; ?></small></span>
      </div>
      <form action="/logout" method="post">
        <input type="hidden" name="_csrf" value="<?= tpl_escape(tpl_get($ctx, 'CSRF')) ?>">
        <button class="logout-button" type="submit">
          <svg viewBox="0 0 24 24"><path d="M10 17l5-5-5-5M15 12H3M15 4h4a2 2 0 0 1 2 2v12a2 2 0 0 1-2 2h-4"/></svg>
          Keluar
        </button>
      </form>
    </div>
  </aside>

  <div class="app-shell">
    <header class="topbar">
      <button class="icon-button mobile-menu" type="button" data-sidebar-toggle aria-label="Buka menu">
        <svg viewBox="0 0 24 24"><path d="M4 7h16M4 12h16M4 17h16"/></svg>
      </button>
      <div class="page-heading">
        <p class="eyebrow">KPU Bea dan Cukai Tipe A Tanjung Priok</p>
        <h1><?= tpl_escape(tpl_get($ctx, 'Title')) ?></h1>
      </div>
      <div class="topbar-actions">
        <div class="topbar-menu">
          <button class="icon-button notification-button" type="button" aria-label="Buka notifikasi" aria-haspopup="true" aria-expanded="false" data-popover-toggle="notifications">
            <svg viewBox="0 0 24 24"><path d="M18 8a6 6 0 0 0-12 0c0 7-3 7-3 9h18c0-2-3-2-3-9M10 21h4"/></svg>
            <?php if (tpl_truthy(tpl_gt(tpl_get($ctx, 'NotificationCount'), 0))): ?><span class="notification-badge"><?= tpl_escape(tpl_get($ctx, 'NotificationCount')) ?></span><?php endif; ?>
          </button>
          <section class="topbar-popover notification-popover" data-popover="notifications" aria-label="Daftar notifikasi" hidden>
            <header class="popover-header">
              <div><p class="eyebrow">Pusat informasi</p><strong>Notifikasi</strong></div>
              <?php if (tpl_truthy(tpl_gt(tpl_get($ctx, 'NotificationCount'), 0))): ?><span class="popover-count"><?= tpl_escape(tpl_get($ctx, 'NotificationCount')) ?> perhatian</span><?php endif; ?>
            </header>
            <div class="notification-list">
              <?php if (tpl_truthy(tpl_get($ctx, 'Notifications'))): ?>
                <?php $__range1 = tpl_iter(tpl_get($ctx, 'Notifications')); if (count($__range1) > 0): $__parent1 = $ctx; foreach ($__range1 as $__key1 => $__item1): $ctx = $__item1; ?>
                <a class="notification-item" href="<?= tpl_escape(tpl_get($ctx, 'URL')) ?>">
                  <span class="notification-tone <?= tpl_escape(tpl_get($ctx, 'Tone')) ?>" aria-hidden="true"></span>
                  <span><strong><?= tpl_escape(tpl_get($ctx, 'Title')) ?></strong><small><?= tpl_escape(tpl_get($ctx, 'Message')) ?></small></span>
                  <svg viewBox="0 0 24 24" aria-hidden="true"><path d="m9 18 6-6-6-6"/></svg>
                </a>
                <?php $ctx = $__parent1; endforeach; endif; ?>
              <?php else: ?>
                <div class="notification-empty">
                  <span class="notification-empty-icon"><svg viewBox="0 0 24 24"><path d="m5 12 4 4L19 6"/></svg></span>
                  <strong>Tidak ada perhatian baru</strong>
                  <small>Pendaftaran, umur barang, dan proses penyelesaian dalam kondisi terkendali.</small>
                </div>
              <?php endif; ?>
            </div>
          </section>
        </div>

        <div class="topbar-menu profile-menu-wrap">
          <button class="user-chip user-chip-button" type="button" aria-label="Buka menu akun" aria-haspopup="true" aria-expanded="false" data-popover-toggle="account-menu">
            <span class="avatar"><?= tpl_escape(tpl_initials(tpl_get($ctx, 'User.DisplayName'))) ?></span>
            <span class="user-copy"><strong><?= tpl_escape(tpl_get($ctx, 'User.DisplayName')) ?></strong><small><?php if (tpl_truthy(tpl_eq(tpl_get($ctx, 'User.Role'), 'admin'))): ?>Administrator<?php else: ?><?php if (tpl_truthy(tpl_get($ctx, 'User.RoleName'))): ?><?= tpl_escape(tpl_get($ctx, 'User.RoleName')) ?><?php else: ?>Petugas TPP<?php endif; ?><?php endif; ?></small></span>
            <svg class="user-chevron" viewBox="0 0 24 24" aria-hidden="true"><path d="m7 10 5 5 5-5"/></svg>
          </button>
          <section class="topbar-popover account-popover" data-popover="account-menu" aria-label="Menu profil" hidden>
            <div class="account-summary">
              <span class="avatar large-avatar"><?= tpl_escape(tpl_initials(tpl_get($ctx, 'User.DisplayName'))) ?></span>
              <span><strong><?= tpl_escape(tpl_get($ctx, 'User.DisplayName')) ?></strong><small><?php if (tpl_truthy(tpl_get($ctx, 'User.Email'))): ?><?= tpl_escape(tpl_get($ctx, 'User.Email')) ?><?php else: ?>Email tidak tersedia<?php endif; ?></small></span>
            </div>
            <div class="account-menu-list">
              <button type="button" class="account-menu-item" data-open-profile-modal>
                <svg viewBox="0 0 24 24"><circle cx="12" cy="8" r="4"/><path d="M4 21a8 8 0 0 1 16 0"/></svg>
                <span><strong>Buka profil</strong><small>Lihat akun dan hak akses</small></span>
              </button>
              <form action="/logout" method="post">
                <input type="hidden" name="_csrf" value="<?= tpl_escape(tpl_get($ctx, 'CSRF')) ?>">
                <button type="submit" class="account-menu-item danger">
                  <svg viewBox="0 0 24 24"><path d="M10 17l5-5-5-5M15 12H3M15 4h4a2 2 0 0 1 2 2v12a2 2 0 0 1-2 2h-4"/></svg>
                  <span><strong>Logout</strong><small>Keluar dari akun ini</small></span>
                </button>
              </form>
            </div>
          </section>
        </div>
      </div>
    </header>

    <main class="content">
      <?php if (tpl_truthy(tpl_get($ctx, 'Success'))): ?><div class="alert success" role="status"><svg viewBox="0 0 24 24"><path d="m5 12 4 4L19 6"/></svg><span><?= tpl_escape(tpl_get($ctx, 'Success')) ?></span><button type="button" data-dismiss aria-label="Tutup">×</button></div><?php endif; ?>
      <?php if (tpl_truthy(tpl_get($ctx, 'Error'))): ?><div class="alert error" role="alert"><svg viewBox="0 0 24 24"><path d="M12 8v5M12 17h.01"/><circle cx="12" cy="12" r="9"/></svg><span><?= tpl_escape(tpl_get($ctx, 'Error')) ?></span><button type="button" data-dismiss aria-label="Tutup">×</button></div><?php endif; ?>
      <?= $content ?>
    </main>
  </div>

  <div class="modal" id="profile-modal" role="dialog" aria-modal="true" aria-labelledby="profile-modal-title" hidden>
    <div class="modal-backdrop" data-close-modal></div>
    <section class="modal-panel modal-panel-small profile-modal-panel">
      <header class="modal-header">
        <div><p class="eyebrow">Akun pengguna</p><h2 id="profile-modal-title">Detail Profil</h2><p>Informasi identitas, role, dan akses akun aktif.</p></div>
        <button class="icon-button" type="button" data-close-modal aria-label="Tutup"><svg viewBox="0 0 24 24"><path d="m6 6 12 12M18 6 6 18"/></svg></button>
      </header>
      <div class="profile-modal-content">
        <div class="profile-hero">
          <span class="avatar profile-avatar"><?= tpl_escape(tpl_initials(tpl_get($ctx, 'User.DisplayName'))) ?></span>
          <div><strong><?= tpl_escape(tpl_get($ctx, 'User.DisplayName')) ?></strong><small><?php if (tpl_truthy(tpl_eq(tpl_get($ctx, 'User.Role'), 'admin'))): ?>Administrator<?php else: ?><?php if (tpl_truthy(tpl_get($ctx, 'User.RoleName'))): ?><?= tpl_escape(tpl_get($ctx, 'User.RoleName')) ?><?php else: ?>Petugas TPP<?php endif; ?><?php endif; ?></small></div>
          <span class="profile-status"><i></i> Aktif</span>
        </div>
        <div class="profile-detail-grid">
          <div><span>Nama lengkap</span><strong><?= tpl_escape(tpl_get($ctx, 'User.DisplayName')) ?></strong></div>
          <div><span>Email</span><strong><?php if (tpl_truthy(tpl_get($ctx, 'User.Email'))): ?><?= tpl_escape(tpl_get($ctx, 'User.Email')) ?><?php else: ?>—<?php endif; ?></strong></div>
          <div><span>Role</span><strong><?php if (tpl_truthy(tpl_eq(tpl_get($ctx, 'User.Role'), 'admin'))): ?>Administrator<?php else: ?><?php if (tpl_truthy(tpl_get($ctx, 'User.RoleName'))): ?><?= tpl_escape(tpl_get($ctx, 'User.RoleName')) ?><?php else: ?>Petugas TPP<?php endif; ?><?php endif; ?></strong></div>
          <div><span>Keamanan sesi</span><strong>Logout otomatis 30 menit</strong></div>
        </div>
        <section class="profile-access-section">
          <div class="section-heading compact-heading"><div><p class="eyebrow">Hak akses</p><h3>Akses yang diberikan</h3></div></div>
          <?php if (tpl_truthy(tpl_eq(tpl_get($ctx, 'User.Role'), 'admin'))): ?>
            <div class="full-access-card"><svg viewBox="0 0 24 24"><path d="M12 3 4 7v5c0 5 3.4 8 8 9 4.6-1 8-4 8-9V7z"/><path d="m9 12 2 2 4-4"/></svg><span><strong>Akses administrator penuh</strong><small>Dapat mengelola seluruh data, pengguna, role, parameter, serta penghapusan barang.</small></span></div>
          <?php else: ?>
            <div class="permission-chip-list">
              <?php $__range2 = tpl_iter(tpl_get($ctx, 'PermissionDefinitions')); if (count($__range2) > 0): $__parent2 = $ctx; foreach ($__range2 as $__key2 => $__item2): $ctx = $__item2; ?><?php if (tpl_truthy(tpl_can(tpl_get($root, 'User'), tpl_get($ctx, 'Code')))): ?><span class="permission-chip"><?= tpl_escape(tpl_get($ctx, 'Label')) ?></span><?php endif; ?><?php $ctx = $__parent2; endforeach; endif; ?>
            </div>
          <?php endif; ?>
        </section>
      </div>
      <footer class="modal-footer profile-modal-footer">
        <button class="button secondary" type="button" data-close-modal>Tutup</button>
        <form action="/logout" method="post"><input type="hidden" name="_csrf" value="<?= tpl_escape(tpl_get($ctx, 'CSRF')) ?>"><button class="button danger" type="submit"><svg viewBox="0 0 24 24"><path d="M10 17l5-5-5-5M15 12H3M15 4h4a2 2 0 0 1 2 2v12a2 2 0 0 1-2 2h-4"/></svg>Logout</button></form>
      </footer>
    </section>
  </div>

  <div class="modal" id="inventory-detail-modal" role="dialog" aria-modal="true" aria-labelledby="inventory-detail-title" hidden>
    <div class="modal-backdrop" data-close-modal></div>
    <section class="modal-panel modal-panel-medium">
      <header class="modal-header"><div><p class="eyebrow">Detail barang</p><h2 id="inventory-detail-title" data-detail-title>Informasi penetapan</h2><p data-detail-subtitle></p></div><button class="icon-button" type="button" data-close-modal aria-label="Tutup"><svg viewBox="0 0 24 24"><path d="m6 6 12 12M18 6 6 18"/></svg></button></header>
      <div class="detail-content" data-detail-content><div class="loading-state"><span class="spinner"></span>Memuat detail…</div></div>
    </section>
  </div>

  <div class="modal" id="timeline-modal" role="dialog" aria-modal="true" aria-labelledby="timeline-title" hidden>
    <div class="modal-backdrop" data-close-modal></div>
    <section class="modal-panel modal-panel-small">
      <header class="modal-header">
        <div><p class="eyebrow">Jejak audit</p><h2 id="timeline-title">Timeline status</h2></div>
        <button class="icon-button" type="button" data-close-modal aria-label="Tutup"><svg viewBox="0 0 24 24"><path d="m6 6 12 12M18 6 6 18"/></svg></button>
      </header>
      <div class="timeline-summary" data-timeline-summary></div>
      <div class="timeline" data-timeline><div class="loading-state"><span class="spinner"></span>Memuat timeline…</div></div>
    </section>
  </div>
<?php endif; ?>
<script src="/assets/app.js" defer></script>
</body>
</html>
