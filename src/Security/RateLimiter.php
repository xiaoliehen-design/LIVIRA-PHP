<?php
declare(strict_types=1);
namespace Livira\Security;

final class RateLimiter
{
    public function __construct(private readonly string $file) {}
    public function allow(string $key,int $limit,int $window):bool
    {
        $now=time();$fp=fopen($this->file,'c+');if(!$fp)return true;flock($fp,LOCK_EX);$raw=stream_get_contents($fp);$all=json_decode($raw?:'{}',true);if(!is_array($all))$all=[];
        $events=array_values(array_filter((array)($all[$key]??[]),fn($t)=>(int)$t>$now-$window));$allowed=count($events)<$limit;if($allowed)$events[]=$now;$all[$key]=$events;
        ftruncate($fp,0);rewind($fp);fwrite($fp,(string)json_encode($all));fflush($fp);flock($fp,LOCK_UN);fclose($fp);return $allowed;
    }
    public function reset(string $key):void{$fp=fopen($this->file,'c+');if(!$fp)return;flock($fp,LOCK_EX);$all=json_decode(stream_get_contents($fp)?:'{}',true);if(!is_array($all))$all=[];unset($all[$key]);ftruncate($fp,0);rewind($fp);fwrite($fp,(string)json_encode($all));flock($fp,LOCK_UN);fclose($fp);}
}
