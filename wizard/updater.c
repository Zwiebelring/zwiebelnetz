#include "stdlib.h"

int main(int argc, char **argv)
{ 
  setuid(0);
  system("/home/pi/zwiebelnetz/wizard/updater.sh");
  return 0;
}
