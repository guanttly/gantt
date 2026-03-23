import JSEncrypt from 'jsencrypt'
import { getRsaKey } from '@/api/login'

// 获取经过Rsa公钥加密后的值
export async function getRsaVal(originVal: string) {
  const rsaKey = await getRsaKey()
  const encrypt = new JSEncrypt()
  encrypt.setPublicKey(rsaKey)
  const val = encrypt.encrypt(originVal) as string
  return val
}
