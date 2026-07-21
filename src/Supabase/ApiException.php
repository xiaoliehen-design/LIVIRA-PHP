<?php
declare(strict_types=1);
namespace Livira\Supabase;
final class ApiException extends \RuntimeException { public function __construct(string $message, public readonly int $status=500, public readonly string $payload=''){parent::__construct($message,$status);} }
