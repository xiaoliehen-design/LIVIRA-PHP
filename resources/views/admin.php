<section class="page-intro admin-intro">
  <div><p><?= tpl_escape(tpl_get($ctx, 'Subtitle')) ?></p></div>
  <div class="admin-tabs" aria-label="Menu administrasi">
    <?php if (tpl_truthy(tpl_can(tpl_get($ctx, 'User'), 'admin.users'))): ?><a class="<?php if (tpl_truthy(tpl_eq(tpl_get($ctx, 'AdminSection'), 'users'))): ?>active<?php endif; ?>" href="/admin/pendaftaran">Pendaftaran</a><?php endif; ?>
    <?php if (tpl_truthy(tpl_can(tpl_get($ctx, 'User'), 'admin.roles'))): ?><a class="<?php if (tpl_truthy(tpl_eq(tpl_get($ctx, 'AdminSection'), 'roles'))): ?>active<?php endif; ?>" href="/admin/roles">Role & akses</a><?php endif; ?>
    <?php if (tpl_truthy(tpl_can(tpl_get($ctx, 'User'), 'admin.parameters'))): ?><a class="<?php if (tpl_truthy(tpl_eq(tpl_get($ctx, 'AdminSection'), 'parameters'))): ?>active<?php endif; ?>" href="/admin/parameters">Parameter</a><?php endif; ?>
  </div>
</section>

<?php if (tpl_truthy(tpl_eq(tpl_get($ctx, 'AdminSection'), 'users'))): ?>
<section class="admin-summary-grid">
  <article><span>Menunggu keputusan</span><strong><?= tpl_escape(tpl_get($ctx, 'PendingUsers')) ?></strong><small>seluruh pendaftaran pending</small></article>
  <article><span>Siap disetujui</span><strong><?= tpl_escape(tpl_get($ctx, 'VerifiedPendingUsers')) ?></strong><small>OTP email sudah terkonfirmasi</small></article>
  <article><span>Role aktif</span><strong><?= tpl_escape(tpl_len(tpl_get($ctx, 'Roles'))) ?></strong><small>tersedia untuk penetapan</small></article>
</section>

<section class="panel table-panel admin-panel">
  <header class="panel-header"><div><h2>Daftar pendaftaran pengguna</h2><p>Urutan terbaru. Akun hanya dapat masuk setelah OTP email terverifikasi dan admin menetapkan role.</p></div></header>
  <div class="table-wrap">
    <table class="data-table admin-user-table">
      <thead><tr><th>Pendaftar</th><th>Konfirmasi email</th><th>Status persetujuan</th><th>Role saat ini</th><th>Tindakan admin</th></tr></thead>
      <tbody>
      <?php $__range1 = tpl_iter(tpl_get($ctx, 'Users')); if (count($__range1) > 0): $__parent1 = $ctx; foreach ($__range1 as $__key1 => $__item1): $ctx = $__item1; ?><?php $registeredUser = $ctx; ?>
        <tr>
          <td><strong><?= tpl_escape(tpl_get($ctx, 'Name')) ?></strong><small><?= tpl_escape(tpl_get($ctx, 'Email')) ?></small><small>Daftar <?= tpl_escape(tpl_datetime(tpl_get($ctx, 'CreatedAt'))) ?></small></td>
          <td>
            <?php if (tpl_truthy(tpl_get($ctx, 'EmailVerified'))): ?><span class="admin-badge success">OTP terverifikasi</span><small><?= tpl_escape(tpl_datetime(tpl_get($ctx, 'EmailVerifiedAt'))) ?></small>
            <?php else: ?><span class="admin-badge warning">Belum verifikasi OTP</span><small>Belum dapat disetujui</small><?php endif; ?>
          </td>
          <td>
            <?php if (tpl_truthy(tpl_eq(tpl_get($ctx, 'ApprovalStatus'), 'approved'))): ?><span class="admin-badge success">Disetujui</span><small><?= tpl_escape(tpl_datetime(tpl_get($ctx, 'ApprovedAt'))) ?> · <?= tpl_escape(tpl_get($ctx, 'ApprovedBy')) ?></small>
            <?php elseif (tpl_truthy(tpl_eq(tpl_get($ctx, 'ApprovalStatus'), 'rejected'))): ?><span class="admin-badge danger">Ditolak</span><small><?= tpl_escape(tpl_get($ctx, 'RejectionReason')) ?></small>
            <?php else: ?><span class="admin-badge warning">Menunggu admin</span><small>Belum memiliki akses aplikasi</small><?php endif; ?>
          </td>
          <td><strong><?php if (tpl_truthy(tpl_get($ctx, 'RoleName'))): ?><?= tpl_escape(tpl_get($ctx, 'RoleName')) ?><?php else: ?>Belum ditetapkan<?php endif; ?></strong><?php if (tpl_truthy(tpl_get($ctx, 'RoleID'))): ?><small>ID role tersimpan</small><?php else: ?><small>Pilih role saat persetujuan</small><?php endif; ?></td>
          <td class="admin-actions-cell">
            <?php if (tpl_truthy(tpl_eq(tpl_get($ctx, 'ApprovalStatus'), 'pending'))): ?>
              <form class="admin-inline-form" method="post" action="/admin/pendaftaran/<?= tpl_escape(tpl_get($ctx, 'ID')) ?>/approve">
                <input type="hidden" name="_csrf" value="<?= tpl_escape(tpl_get($root, 'CSRF')) ?>">
                <select name="role_id" required <?php if (tpl_truthy(tpl_not(tpl_get($ctx, 'EmailVerified')))): ?>disabled<?php endif; ?>>
                  <option value="">Pilih role</option>
                  <?php $__range2 = tpl_iter(tpl_get($root, 'Roles')); if (count($__range2) > 0): $__parent2 = $ctx; foreach ($__range2 as $__key2 => $__item2): $ctx = $__item2; ?><option value="<?= tpl_escape(tpl_get($ctx, 'ID')) ?>"><?= tpl_escape(tpl_get($ctx, 'Name')) ?></option><?php $ctx = $__parent2; endforeach; endif; ?>
                </select>
                <button class="button primary compact" type="submit" <?php if (tpl_truthy(tpl_not(tpl_get($ctx, 'EmailVerified')))): ?>disabled<?php endif; ?>>Setujui</button>
              </form>
              <form class="admin-inline-form reject" method="post" action="/admin/pendaftaran/<?= tpl_escape(tpl_get($ctx, 'ID')) ?>/reject">
                <input type="hidden" name="_csrf" value="<?= tpl_escape(tpl_get($root, 'CSRF')) ?>">
                <input name="reason" maxlength="300" placeholder="Alasan penolakan" required>
                <button class="button ghost compact" type="submit">Tolak</button>
              </form>
            <?php elseif (tpl_truthy(tpl_eq(tpl_get($ctx, 'ApprovalStatus'), 'approved'))): ?>
              <form class="admin-inline-form role-edit-form" method="post" action="/admin/pendaftaran/<?= tpl_escape(tpl_get($ctx, 'ID')) ?>/role">
                <input type="hidden" name="_csrf" value="<?= tpl_escape(tpl_get($root, 'CSRF')) ?>">
                <select name="role_id" required aria-label="Role baru untuk <?= tpl_escape(tpl_get($ctx, 'Name')) ?>">
                  <?php $__range3 = tpl_iter(tpl_get($root, 'Roles')); if (count($__range3) > 0): $__parent3 = $ctx; foreach ($__range3 as $__key3 => $__item3): $ctx = $__item3; ?><option value="<?= tpl_escape(tpl_get($ctx, 'ID')) ?>" <?php if (tpl_truthy(tpl_eq(tpl_get($ctx, 'ID'), tpl_get($registeredUser, 'RoleID')))): ?>selected<?php endif; ?>><?= tpl_escape(tpl_get($ctx, 'Name')) ?></option><?php $ctx = $__parent3; endforeach; endif; ?>
                </select>
                <button class="button secondary compact" type="submit">Simpan role</button>
              </form>
              <small class="admin-muted">Perubahan role berlaku pada login berikutnya.</small>
	            <?php else: ?>
	              <small class="admin-muted">Pendaftaran ditolak. Role hanya dapat diubah untuk pengguna yang telah disetujui.</small>
	            <?php endif; ?>
	            <form class="admin-delete-user-form" method="post" action="/admin/pendaftaran/<?= tpl_escape(tpl_get($ctx, 'ID')) ?>/delete" data-delete-user-form data-user-name="<?= tpl_escape(tpl_get($ctx, 'Name')) ?>" data-user-email="<?= tpl_escape(tpl_get($ctx, 'Email')) ?>">
	              <input type="hidden" name="_csrf" value="<?= tpl_escape(tpl_get($root, 'CSRF')) ?>">
	              <button class="button danger compact" type="submit"><svg viewBox="0 0 24 24"><path d="M4 7h16M9 7V4h6v3M6 7l1 14h10l1-14M10 11v6M14 11v6"/></svg>Hapus user</button>
	            </form>
	            <small class="admin-delete-note">Menghapus akun login dan data pendaftarannya secara permanen.</small>
	          </td>
        </tr>
      <?php $ctx = $__parent1; endforeach; else: ?>
        <tr><td colspan="5"><div class="empty-state small"><h3>Belum ada pendaftaran</h3><p>Pendaftar baru akan muncul setelah menyelesaikan formulir sign-up.</p></div></td></tr>
      <?php endif; ?>
      </tbody>
    </table>
  </div>
</section>
<?php endif; ?>

<?php if (tpl_truthy(tpl_eq(tpl_get($ctx, 'AdminSection'), 'roles'))): ?>
<section class="admin-two-column">
  <article class="panel admin-form-panel">
    <header class="panel-header"><div><h2>Buat role baru</h2><p>Nama role bebas dan akses dapat dikombinasikan sesuai kebutuhan unit.</p></div></header>
    <form class="admin-form" method="post" action="/admin/roles">
      <input type="hidden" name="_csrf" value="<?= tpl_escape(tpl_get($ctx, 'CSRF')) ?>">
      <div class="form-grid cols-2">
        <label>Nama role <em>*</em><input name="name" maxlength="80" placeholder="Contoh: Petugas Lelang BMMN" required></label>
        <label>Deskripsi<input name="description" maxlength="240" placeholder="Jelaskan cakupan tugas role"></label>
      </div>
      <div class="permission-grid">
        <?php $__range4 = tpl_iter(tpl_get($ctx, 'PermissionDefinitions')); if (count($__range4) > 0): $__parent4 = $ctx; foreach ($__range4 as $__key4 => $__item4): $ctx = $__item4; ?>
          <label class="permission-option"><input type="checkbox" name="permissions[]" value="<?= tpl_escape(tpl_get($ctx, 'Code')) ?>"><span><strong><?= tpl_escape(tpl_get($ctx, 'Label')) ?></strong><small><?= tpl_escape(tpl_get($ctx, 'Group')) ?> · <?= tpl_escape(tpl_get($ctx, 'Description')) ?></small></span></label>
        <?php $ctx = $__parent4; endforeach; endif; ?>
      </div>
      <p class="field-help">Hak input awal dan setiap action inventory sekarang berdiri sendiri. Pilih hanya fungsi yang benar-benar diperlukan; hak lihat inventory dan cakupan barang terkait akan ditambahkan otomatis saat role disimpan.</p>
      <button class="button primary" type="submit">Simpan role baru</button>
    </form>
  </article>

  <aside class="admin-guidance">
    <h3>Contoh konfigurasi</h3>
    <article><strong>Lelang saja</strong><p>Pilih Lihat Lelang, Kelola Lelang, Pencarian Detail, dan cakupan barang yang diperlukan.</p></article>
    <article><strong>Petugas BMMN</strong><p>Pilih Akses BMMN, lalu tentukan action yang boleh dijalankan, misalnya usulan, persetujuan peruntukan, atau pengeluaran barang.</p></article>
    <article><strong>Petugas bongkar/muat</strong><p>Pilih Action bongkar/muat kontainer dan cakupan barang yang diperlukan tanpa memberikan action inventory lainnya.</p></article>
  </aside>
</section>

<section class="admin-role-list">
  <?php $__range5 = tpl_iter(tpl_get($ctx, 'Roles')); if (count($__range5) > 0): $__parent5 = $ctx; foreach ($__range5 as $__key5 => $__item5): $ctx = $__item5; ?><?php $role = $ctx; ?>
  <article class="panel role-card <?php if (tpl_truthy(tpl_not(tpl_get($ctx, 'Active')))): ?>inactive<?php endif; ?>">
    <header><div><span class="admin-badge <?php if (tpl_truthy(tpl_get($ctx, 'Active'))): ?>success<?php else: ?>neutral<?php endif; ?>"><?php if (tpl_truthy(tpl_get($ctx, 'Active'))): ?>Aktif<?php else: ?>Nonaktif<?php endif; ?></span><?php if (tpl_truthy(tpl_get($ctx, 'System'))): ?><span class="admin-badge neutral">Role awal</span><?php endif; ?><span class="admin-badge <?php if (tpl_truthy(tpl_gt(tpl_get($ctx, 'AssignedUsers'), 0))): ?>warning<?php else: ?>neutral<?php endif; ?>"><?= tpl_escape(tpl_get($ctx, 'AssignedUsers')) ?> pengguna</span></div><small>Diperbarui <?= tpl_escape(tpl_datetime(tpl_get($ctx, 'UpdatedAt'))) ?></small></header>
    <form method="post" action="/admin/roles/<?= tpl_escape(tpl_get($ctx, 'ID')) ?>/update">
      <input type="hidden" name="_csrf" value="<?= tpl_escape(tpl_get($root, 'CSRF')) ?>">
      <div class="form-grid cols-2">
        <label>Nama role<input name="name" value="<?= tpl_escape(tpl_get($ctx, 'Name')) ?>" maxlength="80" required></label>
        <label>Deskripsi<input name="description" value="<?= tpl_escape(tpl_get($ctx, 'Description')) ?>" maxlength="240"></label>
      </div>
      <div class="permission-grid compact-grid">
        <?php $__range6 = tpl_iter(tpl_get($root, 'PermissionDefinitions')); if (count($__range6) > 0): $__parent6 = $ctx; foreach ($__range6 as $__key6 => $__item6): $ctx = $__item6; ?>
          <label class="permission-option"><input type="checkbox" name="permissions[]" value="<?= tpl_escape(tpl_get($ctx, 'Code')) ?>" <?php if (tpl_truthy(tpl_has_permission(tpl_get($role, 'Permissions'), tpl_get($ctx, 'Code')))): ?>checked<?php endif; ?>><span><strong><?= tpl_escape(tpl_get($ctx, 'Label')) ?></strong><small><?= tpl_escape(tpl_get($ctx, 'Group')) ?></small></span></label>
        <?php $ctx = $__parent6; endforeach; endif; ?>
      </div>
      <div class="role-card-actions"><button class="button secondary compact" type="submit">Simpan perubahan</button></div>
    </form>
    <div class="role-management-actions">
      <form method="post" action="/admin/roles/<?= tpl_escape(tpl_get($ctx, 'ID')) ?>/status" class="role-status-form">
        <input type="hidden" name="_csrf" value="<?= tpl_escape(tpl_get($root, 'CSRF')) ?>">
        <input type="hidden" name="active" value="<?php if (tpl_truthy(tpl_get($ctx, 'Active'))): ?>false<?php else: ?>true<?php endif; ?>">
        <button class="button ghost compact" type="submit"><?php if (tpl_truthy(tpl_get($ctx, 'Active'))): ?>Nonaktifkan role<?php else: ?>Aktifkan kembali<?php endif; ?></button>
      </form>
      <?php if (tpl_truthy(tpl_eq(tpl_get($ctx, 'AssignedUsers'), 0))): ?>
      <form method="post" action="/admin/roles/<?= tpl_escape(tpl_get($ctx, 'ID')) ?>/delete" class="role-delete-form" data-delete-role-form data-role-name="<?= tpl_escape(tpl_get($ctx, 'Name')) ?>">
        <input type="hidden" name="_csrf" value="<?= tpl_escape(tpl_get($root, 'CSRF')) ?>">
        <button class="button danger compact" type="submit">Hapus role</button>
      </form>
      <?php else: ?>
      <p class="role-usage-note">Tidak dapat dihapus selama masih digunakan <?= tpl_escape(tpl_get($ctx, 'AssignedUsers')) ?> pengguna.</p>
      <?php endif; ?>
    </div>
  </article>
  <?php $ctx = $__parent5; endforeach; else: ?><div class="empty-state"><h3>Belum ada role</h3><p>Buat role pertama melalui formulir di atas.</p></div><?php endif; ?>
</section>
<?php endif; ?>

<?php if (tpl_truthy(tpl_eq(tpl_get($ctx, 'AdminSection'), 'parameters'))): ?>
<section class="admin-two-column parameter-layout">
  <article class="panel admin-form-panel">
    <header class="panel-header"><div><h2>Tambah parameter dropdown</h2><p>Parameter aktif langsung dipakai pada formulir dan validasi backend.</p></div></header>
    <form class="admin-form" method="post" action="/admin/parameters" data-parameter-form>
      <input type="hidden" name="_csrf" value="<?= tpl_escape(tpl_get($ctx, 'CSRF')) ?>">
      <div class="form-grid cols-2">
        <label>Kelompok parameter <em>*</em><select name="group_code" required data-parameter-group>
          <option value="">Pilih kelompok</option>
          <option value="bdn_category">Kategori BDN</option>
          <option value="item_kind">Jenis barang</option>
          <option value="goods_condition">Kondisi barang</option>
          <option value="unit">Satuan barang</option>
          <option value="allocation_purpose">Jenis peruntukan BMMN</option>
          <option value="origin_tps">TPS asal</option>
          <option value="tpp">Nama TPP</option>
          <option value="load_type">Jenis muatan (FCL/LCL)</option>
          <option value="exit_type">Jenis pengeluaran</option>
          <option value="transfer_type">Jenis serah terima (Hibah/PSP)</option>
        </select></label>
        <label>Label yang tampil <em>*</em><input name="label" maxlength="160" placeholder="Contoh: Barang mudah rusak" required data-parameter-label></label>
        <label>Kode teknis<input name="code" maxlength="80" placeholder="Opsional; otomatis dari label" data-parameter-code></label>
        <label>Urutan<input type="number" name="sort_order" min="1" max="9999" value="999"></label>
      </div>
      <fieldset class="parameter-scope" data-parameter-scope hidden><legend>Berlaku untuk jenis inventory</legend><label><input type="checkbox" name="applies_to[]" value="BTD"> BTD</label><label><input type="checkbox" name="applies_to[]" value="BDN"> BDN</label><label><input type="checkbox" name="applies_to[]" value="BMMN"> BMMN</label><label><input type="checkbox" name="applies_to[]" value="TITIPAN"> Barang Titipan</label></fieldset>
      <p class="field-help" data-parameter-help>Pilih kelompok untuk melihat kegunaan parameter.</p>
      <button class="button primary" type="submit">Tambah parameter</button>
    </form>
  </article>
  <aside class="admin-guidance">
    <h3>Pengelolaan master</h3>
    <article><strong>Parameter dinonaktifkan, bukan menghapus riwayat.</strong><p>Nilai lama tetap tersimpan pada barang yang pernah menggunakannya, tetapi tidak muncul pada pilihan baru.</p></article>
    <article><strong>Nama TPP terhubung ke master fasilitas.</strong><p>TPP baru langsung tersedia pada filter dan formulir. Kapasitas yard/shed awal bernilai 0 sampai diatur pada database.</p></article>
    <article><strong>Dropdown inti tidak dapat diubah.</strong><p>Jenis inventory BTD/BDN/BMMN/Barang Titipan, status workflow, hasil lelang, dan status lartas tetap dikunci karena memengaruhi logika proses.</p></article>
  </aside>
</section>

<section class="panel table-panel admin-panel">
  <header class="panel-header parameter-table-header"><div><h2>Daftar parameter sistem</h2><p>Parameter dapat dicari, diedit, dinonaktifkan, dan dipulihkan kapan saja.</p></div><form class="parameter-search-form" method="get" action="/admin/parameters"><label class="search-field"><svg viewBox="0 0 24 24"><circle cx="11" cy="11" r="7"/><path d="m20 20-4-4"/></svg><input type="search" name="q" value="<?= tpl_escape(tpl_get($ctx, 'Query')) ?>" placeholder="Cari kelompok, label, atau cakupan…"></label><button class="button secondary compact" type="submit">Cari</button><?php if (tpl_truthy(tpl_get($ctx, 'Query'))): ?><a class="button ghost compact" href="/admin/parameters">Reset</a><?php endif; ?></form></header>
  <div class="table-wrap">
    <table class="data-table"><thead><tr><th>Kelompok</th><th>Label</th><th>Berlaku untuk</th><th>Status</th><th>Tindakan</th></tr></thead><tbody>
      <?php $__range7 = tpl_iter(tpl_get($ctx, 'Parameters')); if (count($__range7) > 0): $__parent7 = $ctx; foreach ($__range7 as $__key7 => $__item7): $ctx = $__item7; ?><tr>
        <td><strong><?= tpl_escape(tpl_parameter_group_label(tpl_get($ctx, 'GroupCode'))) ?></strong><?php if (tpl_truthy(tpl_get($ctx, 'System'))): ?><small>Parameter awal sistem</small><?php else: ?><small>Ditambahkan admin</small><?php endif; ?></td>
        <td><strong><?= tpl_escape(tpl_get($ctx, 'Label')) ?></strong><small>Urutan <?= tpl_escape(tpl_get($ctx, 'SortOrder')) ?></small></td><td><strong><?php if (tpl_truthy(tpl_get($ctx, 'AppliesTo'))): ?><?= tpl_escape(tpl_get($ctx, 'AppliesTo')) ?><?php else: ?>—<?php endif; ?></strong></td>
        <td><span class="admin-badge <?php if (tpl_truthy(tpl_get($ctx, 'Active'))): ?>success<?php else: ?>neutral<?php endif; ?>"><?php if (tpl_truthy(tpl_get($ctx, 'Active'))): ?>Aktif<?php else: ?>Nonaktif<?php endif; ?></span></td>
        <td class="parameter-actions-cell">
          <details class="parameter-edit-details">
            <summary class="button secondary compact">Edit</summary>
            <form class="parameter-edit-form" method="post" action="/admin/parameters/<?= tpl_escape(tpl_get($ctx, 'ID')) ?>/update">
              <input type="hidden" name="_csrf" value="<?= tpl_escape(tpl_get($root, 'CSRF')) ?>">
              <p><strong><?= tpl_escape(tpl_parameter_group_label(tpl_get($ctx, 'GroupCode'))) ?></strong><small>Identitas teknis tetap dipertahankan agar data lama konsisten.</small></p>
              <label>Label<input name="label" value="<?= tpl_escape(tpl_get($ctx, 'Label')) ?>" maxlength="160" required></label>
              <label>Urutan<input type="number" name="sort_order" min="1" max="9999" value="<?= tpl_escape(tpl_get($ctx, 'SortOrder')) ?>" required></label>
              <fieldset><legend>Berlaku untuk</legend><label><input type="checkbox" name="applies_to[]" value="BTD" <?php if (tpl_truthy(tpl_applies_to(tpl_get($ctx, 'AppliesTo'), 'BTD'))): ?>checked<?php endif; ?>> BTD</label><label><input type="checkbox" name="applies_to[]" value="BDN" <?php if (tpl_truthy(tpl_applies_to(tpl_get($ctx, 'AppliesTo'), 'BDN'))): ?>checked<?php endif; ?>> BDN</label><label><input type="checkbox" name="applies_to[]" value="BMMN" <?php if (tpl_truthy(tpl_applies_to(tpl_get($ctx, 'AppliesTo'), 'BMMN'))): ?>checked<?php endif; ?>> BMMN</label><label><input type="checkbox" name="applies_to[]" value="TITIPAN" <?php if (tpl_truthy(tpl_applies_to(tpl_get($ctx, 'AppliesTo'), 'TITIPAN'))): ?>checked<?php endif; ?>> Barang Titipan</label></fieldset>
              <button class="button primary compact" type="submit">Simpan perubahan</button>
            </form>
          </details>
          <form method="post" action="/admin/parameters/<?= tpl_escape(tpl_get($ctx, 'ID')) ?>/status"><input type="hidden" name="_csrf" value="<?= tpl_escape(tpl_get($root, 'CSRF')) ?>"><input type="hidden" name="active" value="<?php if (tpl_truthy(tpl_get($ctx, 'Active'))): ?>false<?php else: ?>true<?php endif; ?>"><button class="button ghost compact" type="submit"><?php if (tpl_truthy(tpl_get($ctx, 'Active'))): ?>Hapus dari dropdown<?php else: ?>Aktifkan kembali<?php endif; ?></button></form>
        </td>
      </tr><?php $ctx = $__parent7; endforeach; else: ?><tr><td colspan="5"><div class="empty-state small"><h3>Parameter belum tersedia</h3></div></td></tr><?php endif; ?>
    </tbody></table>
  </div>
</section>
<?php endif; ?>
