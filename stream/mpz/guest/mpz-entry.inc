uint32_t mpz_decode(const uint8_t *input,uint32_t size,uint8_t *output,uint32_t capacity){
 State *s=calloc(1,sizeof(*s));if(!s)return 0;
 s->memory=calloc(1,MEMORY_SIZE);s->data=malloc(sizeof(tables));
 if(!s->memory||!s->data){free(s->memory);free(s->data);free(s);return 0;}
 memcpy(s->data,tables,sizeof(tables));s->input=input;s->inlen=size;s->output=output;s->outcap=capacity;s->heap=0x10000;esp=MEMORY_SIZE-0x1000;
 uint32_t result=0;
 {
  push(s,0);f_100158cc(s);
  uint32_t state=alloc(s,64);
  push(s,2);push(s,1);push(s,0);ecx=state;f_10016dd0(s);
  if(eax)fail(s,"initialization",eax);
  uint32_t steps=0;
  do{push(s,0);ecx=state;f_100133d0(s);if(++steps>capacity+1u)fail(s,"step limit",steps);}while(eax);
  s->flushing=1;push(s,0);ecx=state;f_10016c7c(s);if(eax)fail(s,"flush",eax);
  if(s->outpos!=capacity)fail(s,"output size mismatch",s->outpos);
  result=s->outpos;
 }
 free(s->memory);free(s->data);free(s);return result;
}
