package dtls

import (
	"fmt"
	"log"
	"os"
)

type handshakeContext interface {
	beginHandshake()
	continueHandshake(*Handshake) (bool, error)
}

type baseHandshakeContext struct {
	*Conn

	isServer                  bool
	nextReceiveSequenceNumber uint16
	sequenceNumber            uint16
	currentFlight             int
	sessionID                 []byte
	cookie                    []byte
	cipherSuite               CipherSuite
	clientRandom              Random
	serverRandom              Random
	keyAgreement              KeyAgreement
	masterSecret              []byte
	finishedHash              finishedHash

	//We omit the pre-flight, i.e. HelloVerify because otherwise we would need to keep state
	//defeating the purpos of HelloVerify
	//Flight 1
	clientHello *Handshake

	//Flight 2
	serverHello        *Handshake
	serverCertificate  *Handshake
	serverKeyExchange  *Handshake
	certificateRequest *Handshake
	serverHelloDone    *Handshake

	//Flight 3
	clientCertificate *Handshake
	clientKeyExchange *Handshake
	certificateVerify *Handshake
	clientFinished    *Handshake

	//Flight 4
	serverFinished *Handshake
}

func (hc *baseHandshakeContext) receiveMessage(message *Handshake) {
	// TODO: Buffer out of order messages
	if message.MessageSeq != hc.nextReceiveSequenceNumber {
		log.Printf("Received out of order message, expected %d was %d", hc.nextReceiveSequenceNumber, message.MessageSeq)
		return
	}
	hc.storeMessage(message)
	hc.nextReceiveSequenceNumber += 1
}

func (hc *baseHandshakeContext) storeMessage(message *Handshake) {
	if hc.currentFlight == 1 && message.MsgType == ClientHello {
		hc.clientHello = message
		return
		//TODO: handle out of order handshake messages?
	}
	if hc.currentFlight == 2 {
		switch message.MsgType {
		case ServerHello:
			hc.serverHello = message
		case Certificate:
			hc.serverCertificate = message
		case ServerKeyExchange:
			hc.serverKeyExchange = message
		case CertificateRequest:
			hc.certificateRequest = message
		case ServerHelloDone:
			hc.serverHelloDone = message
		default:
			log.Printf("Unable to store received handshake message!")
			//TODO: how do we handle invalid handshake messages?
		}
		return
	}
	if hc.currentFlight == 3 {
		switch message.MsgType {
		case Certificate:
			hc.clientCertificate = message
		case ClientKeyExchange:
			hc.clientKeyExchange = message
		case CertificateVerify:
			hc.certificateVerify = message
		case Finished:
			hc.clientFinished = message
		default:
			log.Printf("Unable to store received handshake message!")
			//TODO: how do we handle invalid handshake messages?
		}
		return
	}
	if hc.currentFlight == 4 && message.MsgType == Finished {
		hc.serverFinished = message
		//TODO: handle out of order handshake messages?
		return
	}
	log.Printf("Unable to store received handshake message!")
}

func (hc *baseHandshakeContext) buildNextHandshakeMessage(typ HandshakeType, handshakeMessage []byte) *Handshake {
	handshake := &Handshake{
		MsgType:        typ,
		Length:         uint32(len(handshakeMessage)),
		MessageSeq:     hc.sequenceNumber,
		FragmentOffset: 0,
		FragmentLength: uint32(len(handshakeMessage)),
		Fragment:       handshakeMessage,
	}
	hc.sequenceNumber += 1
	return handshake
}

func (hc *baseHandshakeContext) sendHandshakeMessage(message *Handshake) {
	hc.Conn.SendRecord(TypeHandshake, message.Bytes())
}

func logMasterSecret(random, masterSecret []byte, server bool) {
	f, err := os.OpenFile("/home/maufl/.dtls-secrets", os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		log.Printf("Unable to open log file for DTLS master secret: %s", err)
		return
	}
	defer f.Close()
	var prefix string
	if server {
		prefix = "SERVER_RANDOM"
	} else {
		prefix = "CLIENT_RANDOM"
	}
	if _, err = f.WriteString(fmt.Sprintf("%s %x %x\n", prefix, random, masterSecret)); err != nil {
		log.Printf("Unable to write master secret to log file: %s", err)
	}
}