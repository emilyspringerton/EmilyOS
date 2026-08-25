/* fsaclmod_host.h -- real extern declarations for the two host-side
 * symbols fsacl_mod.c's #target/inline-c bodies call into, plus the
 * mod's own two entry points. Same "-include this header before
 * compiling the generated C" pattern PITVIPER's scrollmod_host.h
 * already established for its own single-function case, extended here
 * to two functions (grant/revoke).
 */
#ifndef FSACLMOD_HOST_H
#define FSACLMOD_HOST_H

extern int emilyos_fsacl_grant(char *identity, char *path);
extern int emilyos_fsacl_revoke(char *identity, char *path);
extern int grant_fs(char *identity, char *path);
extern int revoke_fs(char *identity, char *path);

#endif /* FSACLMOD_HOST_H */
