import JSEncrypt from 'jsencrypt'
import client from '@/api/client'

async function getRsaKey(): Promise<string> {
  const res = await client.get<string>('/auth/rsa-key')
  return res.data
}

// 获取经过Rsa公钥加密后的值
export async function getRsaVal(originVal: string) {
  const rsaKey = await getRsaKey()
  const encrypt = new JSEncrypt()
  encrypt.setPublicKey(rsaKey)
  const val = encrypt.encrypt(originVal) as string
  return val
}
