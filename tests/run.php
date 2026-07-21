<?php
declare(strict_types=1);

$basePath = require dirname(__DIR__).'/bootstrap.php';

use Livira\App;
use Livira\Config;
use Livira\Http\Request;
use Livira\Http\Response;
use Livira\Http\Router;
use Livira\Security\Captcha;
use Livira\Supabase\ApiException;
use Livira\Supabase\DemoStore;
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
    $app = new App($appBase, $config);
    $health = $app->handle(new Request('GET', '/healthz'));
    $assert($health->status === 200 && str_contains($health->body, 'LIVIRA PHP'), 'Kernel aplikasi PHP melayani health check');
    $login = $app->handle(new Request('GET', '/login'));
    $assert($login->status === 200 && str_contains($login->body, 'Masuk'), 'Halaman login PHP berhasil dirender');

    $assert(tpl_get(['item_type' => 'BTD'], 'Type') === 'BTD', 'Alias field template Type ke item_type');
    $assert(tpl_get(['pfpd_required' => true], 'PFPDRequired') === true, 'Konversi acronym PascalCase ke snake_case');

    echo "\nLULUS: {$passed} pemeriksaan.\n";
} finally {
    $remove($temp);
}
