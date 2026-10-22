let scheduled=false
let handler=()=>{}
export function requestFrame():void {
  if(typeof window==="undefined"||scheduled)return
  scheduled=true
  window.requestAnimationFrame(()=>{scheduled=false;handler()})
}
export function setFrameHandler(next:()=>void):void {handler=next;requestFrame()}
