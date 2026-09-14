#include <stdint.h>
#include <stdlib.h>
#include <stdio.h>
#include <string.h>
#include <limits.h>
#include "mpz-tables.inc"
#ifdef MPZ_TRACE
#define TRACE(s,a) ((s)->pc=(a))
#else
#define TRACE(s,a) ((void)0)
#endif
#define MEMORY_SIZE (64u<<20)
typedef struct {
 uint32_t r_eax,r_ebx,r_ecx,r_edx,r_esi,r_edi,r_ebp,r_esp;
 uint32_t cf,zf,sf,of,fs0,pc,flushing;
 uint8_t *memory,*data;
 const uint8_t *input;
 uint8_t *output;
 uint32_t inlen,inpos,outcap,outpos,heap;

} State;
#define eax s->r_eax
#define ebx s->r_ebx
#define ecx s->r_ecx
#define edx s->r_edx
#define esi s->r_esi
#define edi s->r_edi
#define ebp s->r_ebp
#define esp s->r_esp
static void fail(State *s,const char *why,uint32_t detail) {
 fprintf(stderr,"mpz: %s %08x in=%u out=%u pc=%08x\n",why,detail,s->inpos,s->outpos,s->pc);
 abort();
}
static inline uint8_t *mem(State *s,uint32_t a,uint32_t n){
 if(a>=0x10024000u && a<=0x10039000u && n<=0x10039000u-a)return s->data+(a-0x10024000u);
 if(a>=0x1000u && a<=MEMORY_SIZE && n<=MEMORY_SIZE-a)return s->memory+a;
 fail(s,"address",a);return NULL;
}
static inline uint32_t rd8(State *s,uint32_t a){return *mem(s,a,1);}
static inline uint32_t rd16(State *s,uint32_t a){uint16_t v;memcpy(&v,mem(s,a,2),2);return v;}
static inline uint32_t rd32(State *s,uint32_t a){uint32_t v;memcpy(&v,mem(s,a,4),4);return v;}
static inline void wr8(State *s,uint32_t a,uint32_t v){*mem(s,a,1)=v;}
static inline void wr16(State *s,uint32_t a,uint32_t v){uint16_t w=v;memcpy(mem(s,a,2),&w,2);}
static inline void wr32(State *s,uint32_t a,uint32_t v){memcpy(mem(s,a,4),&v,4);}
static inline void push(State *s,uint32_t v){esp-=4;wr32(s,esp,v);}
static inline uint32_t pop(State *s){uint32_t v=rd32(s,esp);esp+=4;return v;}
static uint32_t alloc(State *s,uint32_t size){uint32_t a=s->heap;if(size>MEMORY_SIZE/2||a>MEMORY_SIZE/2-size)fail(s,"allocation",size);s->heap=(a+size+15)&~15u;return a;}
static inline uint32_t alu(State *s,uint32_t a,uint32_t b,int width,int op){
 uint32_t mask=width==32?UINT32_MAX:((1u<<width)-1),sign=1u<<(width-1);a&=mask;b&=mask;uint32_t v;
 switch(op){
 case 0:case 5:{uint64_t q=(uint64_t)a+b+(op==5?s->cf:0);v=q&mask;s->cf=(q>>width)!=0;s->of=((~(a^b)&(a^v))&sign)!=0;break;}
 case 1:v=(a-b)&mask;s->cf=a<b;s->of=(((a^b)&(a^v))&sign)!=0;break;
 case 2:v=a^b;s->cf=s->of=0;break;
 case 3:v=a|b;s->cf=s->of=0;break;
 default:v=a&b;s->cf=s->of=0;break;
 }
 s->zf=v==0;s->sf=(v&sign)!=0;return v;
}
static inline uint32_t shift(State *s,uint32_t a,uint32_t n,int op){
 n&=31;if(!n)return a;uint32_t v;
 if(op==2){v=a<<n;s->cf=(a>>(32-n))&1;s->of=((v>>31)^s->cf)&1;}
 else {v=op==0?(uint32_t)((int32_t)a>>n):a>>n;s->cf=(a>>(n-1))&1;s->of=op==1?a>>31:0;}
 s->zf=v==0;s->sf=v>>31;return v;
}
static inline uint32_t shrd(State *s,uint32_t a,uint32_t b,uint32_t n){n&=31;if(!n)return a;uint32_t v=(a>>n)|(b<<(32-n));s->cf=(a>>(n-1))&1;s->of=(a^v)>>31;s->zf=v==0;s->sf=v>>31;return v;}
static void divide(State *s,uint32_t b,int sign){
 uint64_t a=((uint64_t)edx<<32)|eax,q,r;if(!b)fail(s,"zero divisor",0);
 if(sign){int64_t x=(int64_t)a,y=(int32_t)b;if(x==INT64_MIN&&y==-1)fail(s,"division overflow",0);int64_t z=x/y;if(z<INT32_MIN||z>INT32_MAX)fail(s,"division overflow",0);q=z;r=x%y;}
 else{q=a/b;r=a%b;if(q>UINT32_MAX)fail(s,"division overflow",0);}
 eax=q;edx=r;
}
static void hostcall(State *s,uint32_t f){
 uint32_t a=rd32(s,esp+4),b=rd32(s,esp+8),c=rd32(s,esp+12);uint64_t q;
 switch(f){
 case 0x1001938c: break;
 case 0x1001961e:eax=alloc(s,a);break;
 case 0x10019d6c: if(s->inpos>=s->inlen){if(!s->flushing)fail(s,"truncated input",s->inpos);eax=UINT32_MAX;}else eax=s->input[s->inpos++];break;
 case 0x1001960e:if(s->outpos>=s->outcap)fail(s,"output overflow",s->outpos);s->output[s->outpos++]=a;eax=a&255;break;
 case 0x1001975e:{if(!b){eax=0;break;}uint64_t want=(uint64_t)b*c;uint32_t n=want>s->inlen-s->inpos?s->inlen-s->inpos:want;memcpy(mem(s,a,n),s->input+s->inpos,n);s->inpos+=n;eax=n/b;break;}
 case 0x10019d80: q=(((uint64_t)b<<32)|a)*(((uint64_t)rd32(s,esp+16)<<32)|c);eax=q;edx=q>>32;esp+=16;break;
 case 0x10019dc0:{int64_t x=((uint64_t)b<<32)|a,y=((uint64_t)rd32(s,esp+16)<<32)|c;if(!y||(x==INT64_MIN&&y==-1))fail(s,"division overflow",0);q=x/y;eax=q;edx=q>>32;esp+=16;break;}
 case 0x10019e70:esp-=eax;break;
 case 0x10021724:memset(mem(s,a,c),0,c);eax=a;break;
 case 0x10021850:memset(mem(s,a,c),b,c);eax=a;break;
 default:fail(s,"unknown callback",f);
 }
 esp+=4;
}
