<?php
declare(strict_types=1);

use Livira\App;
use Livira\Config;
use Livira\Http\Request;

$basePath = require dirname(__DIR__).'/bootstrap.php';

$config = Config::load($basePath);
$app = new App($basePath, $config);
$app->handle(new Request())->send();
