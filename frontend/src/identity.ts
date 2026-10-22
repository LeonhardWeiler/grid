export interface Identity { id: string; persistent: boolean; repaired: boolean }
type StorageAccess = Pick<Storage,"getItem"|"setItem">

export function loadIdentity(storage: StorageAccess | null, randomId: () => string): Identity {
  try {
    const stored = storage?.getItem("clientId")
    if (stored && /^[a-zA-Z0-9-]{1,64}$/.test(stored)) return {id:stored,persistent:true,repaired:false}
    const id=randomId()
    storage?.setItem("clientId",id)
    return {id,persistent:storage!==null,repaired:!!stored}
  } catch {
    return {id:randomId(),persistent:false,repaired:false}
  }
}

export function browserIdentity(): Identity {
  let storage: StorageAccess | null=null
  try { storage=window.localStorage } catch { /* Ephemeral identity still works. */ }
  return loadIdentity(storage,()=>crypto.randomUUID())
}
