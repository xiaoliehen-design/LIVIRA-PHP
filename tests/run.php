<?php
declare(strict_types=1);

$basePath = require dirname(__DIR__).'/bootstrap.php';

use Livira\App;
use Livira\Config;
use Livira\Http\Request;
use Livira\Http\Response;
use Livira\Http\Router;
use Livira\Security\Captcha;
use Livira\Security\SessionManager;
use Livira\Supabase\ApiException;
use Livira\Supabase\DemoStore;
use Livira\Supabase\Store;
use Livira\Supabase\SupabaseClient;
use Livira\Support\Xlsx;

$passed = 0;
$assert = static function (bool $condition, string $message) use (&$passed): void {
    if (!$condition) {
        throw new RuntimeException('GAGAL: '.$message);
    }
    $passed++;
    echo "[OK] {$message}\n";
};
$remove = static function (string $dir) use (&$remove): void {
    if (!is_dir($dir)) return;
    foreach (scandir($dir) ?: [] as $entry) {
        if ($entry === '.' || $entry === '..') continue;
        $path = $dir.'/'.$entry;
        if (is_link($path) || is_file($path)) {
            @unlink($path);
        } elseif (is_dir($path)) {
            $remove($path);
        }
    }
    @rmdir($dir);
};

$temp = sys_get_temp_dir().'/livira-php-tests-'.bin2hex(random_bytes(6));
mkdir($temp.'/documents', 0775, true);

try {
    $router = new Router();
    $router->get('/barang/{id}', static fn(Request $r): Response => Response::json(['id' => $r->route('id')]));
    $routeResponse = $router->dispatch(new Request('GET', '/barang/inv-123'));
    $assert($routeResponse->status === 200 && str_contains($routeResponse->body, 'inv-123'), 'Router parameter dinamis');

    $captcha = new Captcha(str_repeat('s', 48), $temp.'/cache');
    [$token, $answer, $expires] = $captcha->challenge();
    $assert($token !== '' && $expires > time(), 'CAPTCHA menghasilkan challenge');
    $assert($captcha->verify($token, $answer), 'CAPTCHA valid hanya dengan jawaban benar');
    $assert(!$captcha->verify($token, $answer), 'CAPTCHA bersifat sekali pakai');

    $store = new DemoStore($temp.'/demo.json', $temp.'/documents');
    $assert(count($store->listInventory(['include_inactive' => true])) >= 4, 'DemoStore memuat seluruh tipe inventory');
    $new = $store->createInventory([
        'type' => 'BTD', 'determination_no' => 'BTD-TEST-001', 'determination_date' => '2026-07-21',
        'description' => 'Barang pengujian', 'quantity' => 2, 'unit' => 'KOLI', 'load_type' => 'FCL',
        'container_no' => 'TEST-123456-7', 'container_size' => '20', 'facility_id' => 'tpp-transporindo',
        'actor' => 'Test Runner',
    ]);
    $assert(($new['container_no'] ?? '') === 'TEST1234567', 'Nomor kontainer dinormalisasi tanpa spasi/tanda hubung');
    $store->addInventoryEvent((string)$new['id'], ['code' => 'pencacahan', 'document_no' => 'BA-CACAH-1', 'document_date' => '2026-07-21', 'actor' => 'Test Runner']);
    $assert(($store->getInventory((string)$new['id'])['status_code'] ?? '') === 'pencacahan', 'Action inventory memperbarui status sesuai tahapan');

    $censusRows = $store->applyCensus((string)$new['id'], [
        [
            'inventory_id' => (string)$new['id'], 'description' => 'Barang pengujian setelah cacah',
            'item_kind' => 'Barang Umum', 'goods_value' => '125.000', 'quantity' => 2,
            'quantity_detail' => '2 koli', 'unit' => 'KOLI', 'goods_condition' => 'Baik',
        ],
        [
            'inventory_id' => '', 'description' => 'Uraian baru hasil pencacahan',
            'item_kind' => 'Mesin', 'goods_value' => '250.000', 'quantity' => 3,
            'quantity_detail' => '3 unit', 'unit' => 'UNIT', 'goods_condition' => 'Rusak ringan',
        ],
    ], ['document_no' => 'BA-CACAH-2', 'document_date' => '2026-07-21', 'actor' => 'Test Runner']);
    $physicalRows = array_values(array_filter(
        $store->listInventory(['include_inactive' => true]),
        static fn(array $row): bool => ($row['physical_unit_id'] ?? '') === ($new['physical_unit_id'] ?? '')
    ));
    $assert(count($censusRows) === 2 && count($physicalRows) === 2, 'Pencacahan FCL menyimpan seluruh uraian lama dan uraian baru');
    $assert(count(array_filter($physicalRows, static fn(array $row): bool => !empty($row['occupancy_primary']))) === 1, 'Uraian baru pencacahan tidak menambah perhitungan okupansi kontainer');

    $emptyRole = $store->createRole(['name' => 'Role Kosong', 'permissions' => ['dashboard.view'], 'actor' => 'Test Runner']);
    $store->deleteRole((string)$emptyRole['id']);
    $assert(true, 'Role tanpa pengguna dapat dihapus');
    try {
        $store->deleteRole('role-operator');
        $assert(false, 'Role terpakai seharusnya tidak dapat dihapus');
    } catch (ApiException $e) {
        $assert($e->getCode() === 409, 'Role dengan pengguna ditolak saat dihapus');
    }

    $xlsx = Xlsx::write(['Nomor', 'Uraian', 'Nilai'], [['A-1', 'Barang uji', 125000]], 'Validasi');
    $xlsxFile = $temp.'/validasi.xlsx';
    file_put_contents($xlsxFile, $xlsx);
    $rows = Xlsx::read($xlsxFile, 10);
    $assert(($rows[0][0] ?? '') === 'Nomor' && ($rows[1][1] ?? '') === 'Barang uji', 'XLSX ekspor dapat dibaca kembali');

    $multiSheet = Xlsx::writeSheets([
        ['name' => 'Ringkasan', 'headers' => ['Indikator', 'Jumlah'], 'rows' => [['Selesai', 1]]],
        ['name' => 'Rincian', 'headers' => ['Dokumen', 'Status'], 'rows' => [['BA-1', 'Selesai']]],
    ]);
    $multiFile = $temp.'/multi-sheet.xlsx';
    file_put_contents($multiFile, $multiSheet);
    $multiExtract = $temp.'/multi-sheet';
    mkdir($multiExtract, 0775, true);
    if (class_exists('ZipArchive')) {
        $zipArchive = new ZipArchive();
        $zipArchive->open($multiFile);
        $zipArchive->extractTo($multiExtract);
        $zipArchive->close();
    } else {
        exec('unzip -qq '.escapeshellarg($multiFile).' -d '.escapeshellarg($multiExtract), $ignored, $unzipCode);
        if ($unzipCode !== 0) throw new RuntimeException('Gagal membuka XLSX multi-sheet.');
    }
    $workbookXml = (string) file_get_contents($multiExtract.'/xl/workbook.xml');
    $assert(is_file($multiExtract.'/xl/worksheets/sheet2.xml') && str_contains($workbookXml, 'Ringkasan') && str_contains($workbookXml, 'Rincian'), 'XLSX performa mendukung sheet Ringkasan dan Rincian');

    $appBase = $temp.'/app';
    mkdir($appBase.'/storage', 0775, true);
    symlink($basePath.'/resources', $appBase.'/resources');
    $config = new Config('LIVIRA', 'development', 'http://localhost', str_repeat('x', 48), 'admin', 'admin-demo-only', '', '', '', 'livira-documents', true, 1800);
    $queryStore = new Store($config, new SupabaseClient('http://127.0.0.1:9', 'test-key', 1));
    $dispositionQuery = $queryStore->dispositionQuery(['type' => 'musnah', 'allowed_types' => ['BTD', 'BDN'], 'status' => 'active', 'sort' => 'determination_newest']);
    $assert(isset($dispositionQuery['inventory_item_type']) && !isset($dispositionQuery['inventory_type']), 'Query proses memakai kolom view inventory_item_type yang tersedia di Supabase');
    $assert(($dispositionQuery['is_active'] ?? '') === 'eq.true' && !isset($dispositionQuery['status_code']), 'Filter proses aktif memakai is_active, bukan status_code active');
    $auctionQuery = $queryStore->dispositionQuery(['type' => 'lelang', 'sort' => 'value_desc']);
    $assert(($auctionQuery['order'] ?? '') === 'htl_value.desc', 'Urutan nilai lelang memakai HTL pada disposition_details');

    $appSeed = new DemoStore($appBase.'/storage/demo-data.json', $appBase.'/storage/demo-documents');
    $appCensusItem = $appSeed->createInventory([
        'type' => 'BTD', 'determination_no' => 'BTD-APP-CACAH-001', 'determination_date' => '2026-07-21',
        'description' => 'Uraian awal aplikasi', 'item_kind' => 'Barang Umum', 'quantity' => 4, 'unit' => 'KOLI',
        'goods_condition' => 'Baik', 'load_type' => 'FCL', 'container_no' => 'APPC1234567',
        'container_size' => '20', 'facility_id' => 'tpp-transporindo', 'actor' => 'Test Runner',
    ]);
    $auctionItem = $appSeed->createInventory([
        'type' => 'BTD', 'determination_no' => 'BTD-LELANG-001', 'determination_date' => '2026-07-21',
        'description' => 'Barang uji proses lelang', 'item_kind' => 'Barang Umum', 'quantity' => 1, 'unit' => 'UNIT',
        'goods_condition' => 'Baik', 'goods_value' => 500000, 'load_type' => 'LCL', 'estimated_volume_m3' => 1,
        'facility_id' => 'tpp-transporindo', 'actor' => 'Test Runner',
    ]);
    $destructionItem = $appSeed->createInventory([
        'type' => 'BDN', 'determination_no' => 'BDN-MUSNAH-001', 'determination_date' => '2026-07-21',
        'description' => 'Barang uji proses musnah', 'item_kind' => 'Barang Umum', 'quantity' => 2, 'unit' => 'UNIT',
        'goods_condition' => 'Rusak berat', 'goods_value' => 200000, 'load_type' => 'LCL', 'estimated_volume_m3' => 1,
        'facility_id' => 'tpp-transporindo', 'actor' => 'Test Runner',
    ]);
    $grantItem = $appSeed->createInventory([
        'type' => 'BMMN', 'determination_no' => 'BMMN-HIBAH-001', 'determination_date' => '2026-07-21',
        'description' => 'Barang uji proses hibah', 'item_kind' => 'Barang Umum', 'quantity' => 3, 'unit' => 'UNIT',
        'goods_condition' => 'Baik', 'goods_value' => 300000, 'load_type' => 'LCL', 'estimated_volume_m3' => 1,
        'facility_id' => 'tpp-transporindo', 'actor' => 'Test Runner',
    ]);
    $app = new App($appBase, $config);
    $health = $app->handle(new Request('GET', '/healthz'));
    $assert($health->status === 200 && str_contains($health->body, 'LIVIRA PHP'), 'Kernel aplikasi PHP melayani health check');
    $login = $app->handle(new Request('GET', '/login'));
    $assert($login->status === 200 && str_contains($login->body, 'Masuk'), 'Halaman login PHP berhasil dirender');

    $sessionManager = new SessionManager($config);
    $adminSession = $sessionManager->adminSession('admin');
    $sessionCookieHeader = $sessionManager->cookie($adminSession);
    preg_match('/^'.preg_quote(SessionManager::COOKIE, '/').'=([^;]+)/', $sessionCookieHeader, $cookieMatch);
    $_COOKIE[SessionManager::COOKIE] = $cookieMatch[1] ?? '';
    $inventoryPage = $app->handle(new Request('GET', '/inventory'));
    $assert(
        $inventoryPage->status === 200 && str_contains($inventoryPage->body, 'data-target-id="'.(string)$appCensusItem['id'].'"'),
        'Target pencacahan menggunakan ID inventory utama, bukan physical_unit_id'
    );
    $assert(str_contains($inventoryPage->body, '/templates/template_upload_btd.xlsx?v=1.0.6'), 'Tautan template BTD mengarah ke file publik yang tersedia');
    $assetScript = (string) file_get_contents($basePath.'/public/assets/app.js');
    $assert(str_contains($assetScript, '/templates/template_upload_bdn.xlsx?v=1.0.6'), 'Konfigurasi template BDN mengarah ke file publik yang tersedia');
    foreach (['template_upload_btd.xlsx' => 'BTD', 'template_upload_bdn.xlsx' => 'BDN'] as $templateFile => $templateType) {
        $templatePath = $basePath.'/public/templates/'.$templateFile;
        $compatibilityPath = $basePath.'/public/assets/templates/'.$templateFile;
        $templateRows = Xlsx::read($templatePath, 5);
        $nonBlankTemplateRows = array_values(array_filter($templateRows, static fn(array $row): bool => count(array_filter($row, static fn($value): bool => trim((string)$value) !== '')) > 0));
        $assert(
            is_file($templatePath)
            && is_file($compatibilityPath)
            && hash_file('sha256', $templatePath) === hash_file('sha256', $compatibilityPath)
            && count($nonBlankTemplateRows) === 2,
            'Template '.$templateType.' tersedia identik pada URL utama dan kompatibilitas serta hanya memiliki satu baris contoh'
        );
        $beforeImport = (new DemoStore($appBase.'/storage/demo-data.json', $appBase.'/storage/demo-documents'))->countInventory(['include_inactive' => true]);
        $importResponse = $app->handle(new Request('POST', '/inventory/import', [], [
            '_csrf' => $adminSession['CSRF'], 'item_type' => $templateType, 'return_to' => '/inventory',
        ], [
            'excel_file' => [
                'name' => $templateFile, 'tmp_name' => $templatePath, 'size' => filesize($templatePath),
                'error' => UPLOAD_ERR_OK, 'type' => 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet',
            ],
        ], ['accept' => 'text/html']));
        $afterImport = (new DemoStore($appBase.'/storage/demo-data.json', $appBase.'/storage/demo-documents'))->countInventory(['include_inactive' => true]);
        $assert($importResponse->status === 303 && str_contains((string)($importResponse->headers['Location'] ?? ''), 'berhasil') && $afterImport === $beforeImport + 1, 'Template '.$templateType.' konsisten dengan parser dan berhasil diimpor end-to-end');
    }
    $censusPayload = [[
        'target_id' => (string)$appCensusItem['id'],
        'load_type' => 'FCL',
        'lines' => [
            [
                'inventory_id' => (string)$appCensusItem['id'], 'description' => 'Uraian awal diperbarui',
                'item_kind' => 'Barang Umum', 'goods_value' => '100.000', 'quantity' => 4,
                'quantity_detail' => '', 'unit' => 'KOLI', 'goods_condition' => 'Baik',
            ],
            [
                'inventory_id' => '', 'description' => 'Uraian tambahan dari form',
                'item_kind' => 'Mesin', 'goods_value' => '300.000', 'quantity' => 2,
                'quantity_detail' => '2 unit', 'unit' => 'UNIT', 'goods_condition' => 'Rusak ringan',
            ],
        ],
    ]];
    $censusResponse = $app->handle(new Request('POST', '/inventory/bulk-event', [], [
        '_csrf' => $adminSession['CSRF'], 'event_code' => 'pencacahan',
        'document_no' => 'BA-APP-CACAH-001', 'document_date' => '2026-07-21',
        'census_results_json' => json_encode($censusPayload, JSON_UNESCAPED_UNICODE|JSON_UNESCAPED_SLASHES),
        'return_to' => '/inventory',
    ], [], ['accept' => 'text/html']));
    $assert($censusResponse->status === 303 && str_contains((string)($censusResponse->headers['Location'] ?? ''), 'Hasil%20pencacahan'), 'Form pencacahan tidak lagi mensyaratkan inventory_ids dari picker umum');
    $appCheck = new DemoStore($appBase.'/storage/demo-data.json', $appBase.'/storage/demo-documents');
    $appCensusRows = array_values(array_filter(
        $appCheck->listInventory(['include_inactive' => true]),
        static fn(array $row): bool => ($row['physical_unit_id'] ?? '') === ($appCensusItem['physical_unit_id'] ?? '')
    ));
    $assert(count($appCensusRows) === 2 && in_array('Uraian tambahan dari form', array_column($appCensusRows, 'description'), true), 'Handler HTTP menyimpan uraian baru sesuai jumlah baris pencacahan');

    foreach (['lelang', 'musnah', 'hibah'] as $processType) {
        $pageResponse = $app->handle(new Request('GET', '/proses/'.$processType));
        $assert($pageResponse->status === 200 && !str_contains($pageResponse->body, 'Kesalahan'), 'Halaman proses '.$processType.' dapat dirender tanpa query kolom Supabase yang salah');
    }

    $processPost = static function(string $type, array $body) use ($app, $adminSession): Response {
        return $app->handle(new Request('POST', '/proses/'.$type.'/bulk-action', [], array_merge([
            '_csrf' => $adminSession['CSRF'], 'document_date' => '2026-07-21', 'return_to' => '/proses/'.$type,
        ], $body), [], ['accept' => 'text/html']));
    };

    $auctionStart = $processPost('lelang', ['event_code' => 'kep_lelang', 'document_no' => 'KEP-LELANG-001', 'inventory_ids' => [(string)$auctionItem['id']]]);
    $assert($auctionStart->status === 303, 'Proses lelang dapat dimulai dari inventory');
    $processCheck = new DemoStore($appBase.'/storage/demo-data.json', $appBase.'/storage/demo-documents');
    $auctionProcess = $processCheck->listDispositions(['type' => 'lelang', 'inventory_id' => (string)$auctionItem['id'], 'include_inactive_inventory' => true])[0] ?? [];
    $assert(($auctionProcess['status_code'] ?? '') === 'kep_lelang', 'Penerbitan KEP Lelang diterapkan setelah placeholder proses dibuat');

    $auctionId = (string)($auctionProcess['id'] ?? '');
    $htlResponse = $processPost('lelang', ['event_code' => 'kep_htl', 'document_no' => 'KEP-HTL-001', 'process_ids' => [$auctionId], 'htl_results_json' => json_encode([['process_id' => $auctionId, 'htl_value' => '450.000']])]);
    $assert($htlResponse->status === 303, 'Nilai HTL per barang dapat disimpan');
    $scheduleResponse = $processPost('lelang', ['event_code' => 'jadwal_lelang', 'document_no' => 'ND-JADWAL-001', 'process_ids' => [$auctionId], 'execution_start_date' => '2026-07-25', 'execution_end_date' => '']);
    $assert($scheduleResponse->status === 303, 'Penjadwalan lelang dapat disimpan');
    $schedulePage = $app->handle(new Request('GET', '/proses/lelang'));
    $assert(str_contains($schedulePage->body, 'ND-JADWAL-001'), 'Kelompok ND penjadwalan memakai struktur DocumentNo dan Processes yang sesuai view');
    $completionResponse = $processPost('lelang', ['event_code' => 'selesai_lelang', 'document_no' => 'RISALAH-001', 'auction_schedule_no' => 'ND-JADWAL-001', 'auction_results_json' => json_encode([['process_id' => $auctionId, 'outcome' => 'laku', 'sale_value' => '600.000']])]);
    $assert($completionResponse->status === 303, 'Hasil lelang satu ND dapat disimpan lengkap');
    $allocationResponse = $processPost('lelang', ['event_code' => 'alokasi_hasil_lelang', 'document_no' => 'KEP-ALOKASI-001', 'process_ids' => [$auctionId], 'allocation_target' => 'Kas Negara']);
    $assert($allocationResponse->status === 303, 'Alokasi hasil lelang dapat menyelesaikan proses');

    $musnahStart = $processPost('musnah', ['event_code' => 'kep_musnah', 'document_no' => 'KEP-MUSNAH-001', 'inventory_ids' => [(string)$destructionItem['id']], 'destruction_cost' => '125.000']);
    $assert($musnahStart->status === 303, 'KEP Musnah dapat membuat dan memperbarui proses');
    $processCheck = new DemoStore($appBase.'/storage/demo-data.json', $appBase.'/storage/demo-documents');
    $musnahProcess = $processCheck->listDispositions(['type' => 'musnah', 'inventory_id' => (string)$destructionItem['id'], 'include_inactive_inventory' => true])[0] ?? [];
    $assert(($musnahProcess['status_code'] ?? '') === 'kep_musnah' && (int)($musnahProcess['destruction_cost'] ?? 0) === 125000, 'Biaya KEP Musnah tersimpan pada proses');
    $musnahId = (string)($musnahProcess['id'] ?? '');
    $musnahFinish = $processPost('musnah', ['event_code' => 'ba_musnah', 'document_no' => 'BA-MUSNAH-001', 'process_ids' => [$musnahId], 'destruction_cost' => '150.000']);
    $assert($musnahFinish->status === 303, 'BA Musnah dapat menyelesaikan proses');

    $grantStart = $processPost('hibah', ['event_code' => 'ba_serah_terima', 'document_no' => 'BA-HIBAH-001', 'inventory_ids' => [(string)$grantItem['id']], 'transfer_type' => 'hibah']);
    $assert($grantStart->status === 303, 'BA Serah Terima dapat membuat dan menyelesaikan proses Hibah/PSP');
    $processCheck = new DemoStore($appBase.'/storage/demo-data.json', $appBase.'/storage/demo-documents');
    $grantProcess = $processCheck->listDispositions(['type' => 'hibah', 'inventory_id' => (string)$grantItem['id'], 'include_inactive_inventory' => true])[0] ?? [];
    $assert(($grantProcess['status_code'] ?? '') === 'ba_serah_terima' && ($grantProcess['transfer_type'] ?? '') === 'hibah' && empty($grantProcess['is_active']), 'Status akhir Hibah/PSP dan jenis serah terima tersimpan');

    $logout = $app->handle(new Request('POST', '/logout', [], ['_csrf' => $adminSession['CSRF']], [], ['accept' => 'text/html']));
    $logoutCookie = (string)($logout->headers['Set-Cookie'] ?? '');
    $assert($logout->status === 303 && str_starts_with((string)($logout->headers['Location'] ?? ''), '/login'), 'Logout form mengarahkan ke halaman login');
    $assert(str_starts_with($logoutCookie, SessionManager::COOKIE.'=;') && str_contains($logoutCookie, 'Max-Age=0'), 'Logout menghapus cookie dan tidak ditimpa middleware sesi');

    $_COOKIE[SessionManager::COOKIE] = $cookieMatch[1] ?? '';
    $idleLogout = $app->handle(new Request('POST', '/session/idle-logout', [], [], [], ['accept' => 'application/json', 'x-csrf-token' => (string)$adminSession['CSRF']]));
    $idleCookie = (string)($idleLogout->headers['Set-Cookie'] ?? '');
    $assert($idleLogout->status === 200 && str_contains($idleLogout->body, '"ok":true') && str_starts_with($idleCookie, SessionManager::COOKIE.'=;'), 'Idle logout menerima CSRF header dan memutus sesi');
    unset($_COOKIE[SessionManager::COOKIE]);

    $assert(tpl_get(['item_type' => 'BTD'], 'Type') === 'BTD', 'Alias field template Type ke item_type');
    $assert(tpl_get(['pfpd_required' => true], 'PFPDRequired') === true, 'Konversi acronym PascalCase ke snake_case');

    echo "\nLULUS: {$passed} pemeriksaan.\n";
} finally {
    $remove($temp);
}
