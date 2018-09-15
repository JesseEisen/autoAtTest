#sleep=3
#port=COM5
#baudrate=115200

send ATI
read [Quectel, EC20F, Revision: EC20CEFAGR06A05M4G, OK]

send AT+GMI
read [Simplight, OK]

send AT+GMI
read [Quectel, OK]

