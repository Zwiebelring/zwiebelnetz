#include "stdlib.h"

int main(int argc, char **argv)
{ 
  setuid(0);
  system("./updater.sh");
  return 0;
}
