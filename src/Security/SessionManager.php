<?php
declare(strict_types=1);
namespace Livira\Security;

use Livira\Config;

final class SessionManager
{
    public const COOKIE='livira_session';
    public function __construct(private readonly Config $config) {}
    public function create(array $session): array
    {
        $now=time();
        $session=array_merge(['Subject'=>'','Email'=>'','DisplayName'=>'','Role'=>'user','RoleID'=>'','RoleName'=>'','Permissions'=>[],'SessionVersion'=>0,'LastActivity'=>$now,'ExpiresAt'=>$now+28800,'CSRF'=>bin2hex(random_bytes(24))],$session);
        return $session;
    }
    public function read(): ?array
    {
        $raw=$_COOKIE[self::COOKIE]??''; if($raw==='')return null;
        $parts=explode('.',$raw); if(count($parts)!==2)return null;
        [$payload,$sig]=$parts; if(!hash_equals($this->sign($payload),$sig))return null;
        $json=$this->b64d($payload); if($json===false)return null;
        $s=json_decode($json,true); if(!is_array($s))return null;
        if((int)($s['ExpiresAt']??0)<time())return null;
        if(time()-(int)($s['LastActivity']??0)>$this->config->idleTimeoutSeconds)return null;
        if(($s['Role']??'')==='admin'&&!$this->validAdmin($s))return null;
        return $s;
    }
    public function touch(array $s): array { $s['LastActivity']=time(); return $s; }
    public function cookie(array $session): string
    {
        $payload=$this->b64e((string)json_encode($session,JSON_UNESCAPED_UNICODE|JSON_UNESCAPED_SLASHES));
        $secure=$this->config->production()||str_starts_with($this->config->publicBaseUrl,'https://');
        return self::COOKIE.'='.$payload.'.'.$this->sign($payload).'; Path=/; Max-Age='.max(0,(int)$session['ExpiresAt']-time()).'; HttpOnly; SameSite=Lax'.($secure?'; Secure':'');
    }
    public function clearCookie(): string { return self::COOKIE.'=; Path=/; Max-Age=0; Expires=Thu, 01 Jan 1970 00:00:01 GMT; HttpOnly; SameSite=Lax'.(($this->config->production()||str_starts_with($this->config->publicBaseUrl,'https://'))?'; Secure':''); }
    public function csrfValid(array $session,string $token): bool { return $token!==''&&hash_equals((string)($session['CSRF']??''),$token); }
    public function adminSession(string $identity): array { return $this->create(['Subject'=>'admin:'.$identity,'DisplayName'=>'Administrator','Role'=>'admin','RoleName'=>'Administrator','Permissions'=>['*'],'SessionVersion'=>$this->adminVersion()]); }
    private function validAdmin(array $s):bool{return $this->config->adminUsername!==''&&hash_equals('admin:'.$this->config->adminUsername,(string)($s['Subject']??''))&&(int)($s['SessionVersion']??0)===$this->adminVersion();}
    private function adminVersion():int{$raw=hash('sha256',$this->config->adminUsername."\0".$this->config->adminPassword,true);$n=unpack('J',substr($raw,0,8))[1]??1;return max(1,(int)($n&PHP_INT_MAX));}
    private function sign(string $v):string{return $this->b64e(hash_hmac('sha256',$v,$this->config->sessionSecret,true));}
    private function b64e(string $v):string{return rtrim(strtr(base64_encode($v),'+/','-_'),'=');}
    private function b64d(string $v):string|false{$pad=strlen($v)%4;if($pad)$v.=str_repeat('=',4-$pad);return base64_decode(strtr($v,'-_','+/'),true);}
}
