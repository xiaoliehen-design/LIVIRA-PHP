<?php
declare(strict_types=1);
namespace Livira\Security;

final class Captcha
{
    public function __construct(private readonly string $secret,private readonly string $cacheDir)
    {
        if(!is_dir($cacheDir)&&!@mkdir($cacheDir,0775,true)&&!is_dir($cacheDir))throw new \RuntimeException('Direktori CAPTCHA tidak dapat dibuat.');
    }
    /** @return array{0:string,1:string,2:int} */
    public function challenge(): array
    {
        $alphabet='ABCDEFGHJKLMNPQRSTUVWXYZ23456789'; $answer='';
        for($i=0;$i<5;$i++)$answer.=$alphabet[random_int(0,strlen($alphabet)-1)];
        $nonce=bin2hex(random_bytes(8));$expires=time()+300;
        $payload=['h'=>hash_hmac('sha256',strtoupper($answer).'|'.$nonce,$this->secret),'n'=>$nonce,'e'=>$expires];
        $raw=$this->b64e((string)json_encode($payload));$token=$raw.'.'.$this->b64e(hash_hmac('sha256',$raw,$this->secret,true));
        $this->remember($token,$answer,$expires);
        return [$token,$answer,$expires];
    }
    public function verify(string $token,string $answer):bool
    {
        $parts=explode('.',$token);if(count($parts)!==2)return false;[$raw,$sig]=$parts;
        if(!hash_equals($this->b64e(hash_hmac('sha256',$raw,$this->secret,true)),$sig))return false;
        $json=$this->b64d($raw);$payload=is_string($json)?json_decode($json,true):null;
        if(!is_array($payload)||(int)($payload['e']??0)<time())return false;
        $remembered=$this->loadAnswer($token);
        if($remembered===null)return false;
        $expected=hash_hmac('sha256',strtoupper(trim($answer)).'|'.($payload['n']??''),$this->secret);
        $valid=hash_equals((string)($payload['h']??''),$expected)&&hash_equals(strtoupper($remembered),strtoupper(trim($answer)));
        if($valid)@unlink($this->file($token));
        return $valid;
    }
    public function image(string $token):string
    {
        $answer=$this->loadAnswer($token)??'ERROR';$chars=str_split($answer);$x=26;$texts='';
        foreach($chars as $i=>$c){$rot=($i%2===0?-6:7);$y=43+($i%3-1)*3;$texts.='<text x="'.$x.'" y="'.$y.'" transform="rotate('.$rot.' '.$x.' '.$y.')">'.htmlspecialchars($c,ENT_QUOTES|ENT_SUBSTITUTE,'UTF-8').'</text>';$x+=31;}
        return '<svg xmlns="http://www.w3.org/2000/svg" width="190" height="62" viewBox="0 0 190 62"><rect width="190" height="62" rx="12" fill="#f5f7fb"/><path d="M4 18L186 45M8 50L182 13M20 8L164 55" stroke="#cbd5e1" stroke-width="2" opacity=".7"/><g fill="#172554" font-family="monospace" font-size="30" font-weight="700">'.$texts.'</g><circle cx="24" cy="49" r="3" fill="#94a3b8"/><circle cx="171" cy="18" r="3" fill="#94a3b8"/></svg>';
    }
    private function remember(string $token,string $answer,int $expires):void{file_put_contents($this->file($token),(string)json_encode(['a'=>$answer,'e'=>$expires]),LOCK_EX);}
    private function loadAnswer(string $token):?string{$file=$this->file($token);if(!is_file($file))return null;$payload=json_decode((string)file_get_contents($file),true);if(!is_array($payload)||(int)($payload['e']??0)<time()){@unlink($file);return null;}return (string)($payload['a']??'');}
    private function file(string $token):string{return $this->cacheDir.'/captcha_'.hash('sha256',$token).'.json';}
    private function b64e(string $value):string{return rtrim(strtr(base64_encode($value),'+/','-_'),'=');}
    private function b64d(string $value):string|false{$pad=strlen($value)%4;if($pad)$value.=str_repeat('=',4-$pad);return base64_decode(strtr($value,'-_','+/'),true);}
}
