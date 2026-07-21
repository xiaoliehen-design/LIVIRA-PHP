<?php
declare(strict_types=1);

$basePath = __DIR__;
date_default_timezone_set('Asia/Jakarta');

require_once $basePath.'/src/Support/helpers.php';

spl_autoload_register(static function (string $class) use ($basePath): void {
    $prefix = 'Livira\\';
    if (!str_starts_with($class, $prefix)) {
        return;
    }
    $relative = substr($class, strlen($prefix));
    $file = $basePath.'/src/'.str_replace('\\', '/', $relative).'.php';
    if (is_file($file)) {
        require_once $file;
    }
});

foreach ([$basePath.'/storage/cache', $basePath.'/storage/logs', $basePath.'/storage/demo-documents'] as $directory) {
    if (!is_dir($directory)) {
        @mkdir($directory, 0775, true);
    }
}

return $basePath;
