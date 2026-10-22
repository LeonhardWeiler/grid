import {test} from "node:test"
import assert from "node:assert/strict"
import {exceedsDragThreshold} from "./gesture.ts"

test("many tiny pointer movements count as a drag after crossing the total threshold",()=>{
  const start={x:10,y:10}
  assert(!exceedsDragThreshold(start,{x:11,y:10}))
  assert(exceedsDragThreshold(start,{x:18,y:10}))
})
