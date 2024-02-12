// Copyright 2019 Jason Ertel (github.com/jertel).
// Copyright 2020-2023 Security Onion Solutions LLC and/or licensed to Security Onion Solutions LLC under one
// or more contributor license agreements. Licensed under the Elastic License 2.0 as shown at
// https://securityonion.net/license; you may not use this file except in compliance with the
// Elastic License 2.0.

package packet

import (
	"bytes"
	"encoding/base64"
	"io"
	"os"

	"github.com/apex/log"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"github.com/google/gopacket/pcapgo"
	"github.com/security-onion-solutions/securityonion-soc/model"
)

var SupportedLayerTypes = [...]gopacket.LayerType{
	layers.LayerTypeARP,
	layers.LayerTypeICMPv4,
	layers.LayerTypeICMPv6,
	layers.LayerTypeIPSecAH,
	layers.LayerTypeIPSecESP,
	layers.LayerTypeNTP,
	layers.LayerTypeSIP,
	layers.LayerTypeTLS,
}

func ParsePcap(filename string, offset int, count int, unwrap bool) ([]*model.Packet, error) {
	packets := make([]*model.Packet, 0)
	parsePcapFile(filename, func(index int, pcapPacket gopacket.Packet) bool {
		if index >= offset {
			packet := model.NewPacket(index)
			parseData(pcapPacket, packet, unwrap)
			packets = append(packets, packet)
		}
		return len(packets) < count
	})
	return packets, nil
}

func ToStream(packets []gopacket.Packet) (io.ReadCloser, error) {
	var snaplen uint32 = 65536
	var full bytes.Buffer

	writer := pcapgo.NewWriter(&full)
	writer.WriteFileHeader(snaplen, layers.LinkTypeEthernet)

	opts := gopacket.SerializeOptions{}

	buf := gopacket.NewSerializeBuffer()
	for _, packet := range packets {
		buf.Clear()
		err := gopacket.SerializePacket(buf, opts, packet)
		if err != nil {
			return nil, err
		}
		writer.WritePacket(packet.Metadata().CaptureInfo, buf.Bytes())
	}
	return io.NopCloser(bytes.NewReader(full.Bytes())), nil
}

func getPacketProtocol(packet gopacket.Packet) string {
	if packet.Layer(layers.LayerTypeTCP) != nil {
		return model.PROTOCOL_TCP
	}
	if packet.Layer(layers.LayerTypeUDP) != nil {
		return model.PROTOCOL_UDP
	}
	if packet.Layer(layers.LayerTypeICMPv4) != nil ||
		packet.Layer(layers.LayerTypeICMPv6) != nil {
		return model.PROTOCOL_ICMP
	}
	return ""
}

func filterPacket(filter *model.Filter, packet gopacket.Packet) bool {
	var srcIp, dstIp string
	var srcPort, dstPort int

	timestamp := packet.Metadata().Timestamp
	layer := packet.Layer(layers.LayerTypeIPv6)
	if layer != nil {
		layer := layer.(*layers.IPv6)
		srcIp = layer.SrcIP.String()
		dstIp = layer.DstIP.String()
	} else {
		layer = packet.Layer(layers.LayerTypeIPv4)
		if layer != nil {
			layer := layer.(*layers.IPv4)
			srcIp = layer.SrcIP.String()
			dstIp = layer.DstIP.String()
		}
	}

	layer = packet.Layer(layers.LayerTypeTCP)
	if layer != nil {
		layer := layer.(*layers.TCP)
		srcPort = int(layer.SrcPort)
		dstPort = int(layer.DstPort)
	}

	layer = packet.Layer(layers.LayerTypeUDP)
	if layer != nil {
		layer := layer.(*layers.UDP)
		srcPort = int(layer.SrcPort)
		dstPort = int(layer.DstPort)
	}

	include := (filter.BeginTime.IsZero() || timestamp.After(filter.BeginTime)) &&
		(filter.EndTime.IsZero() || timestamp.Before(filter.EndTime)) &&
		(filter.Protocol == "" || filter.Protocol == getPacketProtocol(packet)) &&
		(filter.SrcIp == "" || srcIp == filter.SrcIp) &&
		(filter.DstIp == "" || dstIp == filter.DstIp)

	if include && (filter.Protocol == "udp" || filter.Protocol == "tcp") {
		include = (filter.SrcPort == 0 || srcPort == filter.SrcPort) &&
			(filter.DstPort == 0 || dstPort == filter.DstPort)
	}

	return include
}

func ParseRawPcap(filename string, count int, filter *model.Filter) ([]gopacket.Packet, error) {
	packets := make([]gopacket.Packet, 0)
	err := parsePcapFile(filename, func(index int, pcapPacket gopacket.Packet) bool {
		if filterPacket(filter, pcapPacket) {
			packets = append(packets, pcapPacket)
		} else {
			pcapPacket = unwrapVxlanPacket(pcapPacket, nil)
			if filterPacket(filter, pcapPacket) {
				packets = append(packets, pcapPacket)
			}
		}

		return len(packets) < count
	})

	if len(packets) == count {
		log.WithFields(log.Fields{
			"packetCount": len(packets),
		}).Warn("Exceeded packet capture limit for job; returned PCAP will be truncated")
	}

	return packets, err
}

func UnwrapPcap(filename string, unwrappedFilename string) bool {
	unwrapped := false
	info, err := os.Stat(unwrappedFilename)
	if os.IsNotExist(err) {
		unwrappedFile, err := os.Create(unwrappedFilename)
		if err != nil {
			log.WithError(err).WithField("unwrappedFilename", unwrappedFilename).Error("Unable to create unwrapped file")
		} else {
			writer := pcapgo.NewWriter(unwrappedFile)
			err = writer.WriteFileHeader(65535, layers.LinkTypeEthernet)
			if err != nil {
				log.WithError(err).WithField("unwrappedFilename", unwrappedFilename).Error("Unable to write unwrapped file header")
			} else {
				defer unwrappedFile.Close()
				err = parsePcapFile(filename, func(index int, pcapPacket gopacket.Packet) bool {
					newPacket := unwrapVxlanPacket(pcapPacket, nil)
					err = writer.WritePacket(newPacket.Metadata().CaptureInfo, newPacket.Data())
					if err != nil {
						log.WithError(err).WithFields(log.Fields{
							"unwrappedFilename": unwrappedFilename,
							"index":             index,
						}).Error("Unable to write unwrapped file packet")
						return false
					}
					return true
				})
				if err != nil {
					log.WithError(err).WithField("filename", filename).Error("Unable to parse PCAP into unwrapped PCAP")
				} else {
					unwrapped = true
				}
			}
		}
	} else if info.IsDir() {
		log.WithField("unwrappedFilename", unwrappedFilename).Error("Unexpected directory found with unwrapped filename")
	} else {
		unwrapped = true
	}

	return unwrapped

}

func parsePcapFile(filename string, handler func(int, gopacket.Packet) bool) error {
	handle, err := pcap.OpenOffline(filename)
	if err == nil {
		defer handle.Close()
		packetSource := gopacket.NewPacketSource(handle, handle.LinkType())
		packetSource.DecodeOptions.Lazy = true
		packetSource.DecodeOptions.NoCopy = true
		index := 0
		for pcapPacket := range packetSource.Packets() {
			if pcapPacket != nil {
				if !handler(index, pcapPacket) {
					break
				}
				index++
			}
		}
	}
	return err
}

func overrideType(packet *model.Packet, layerType gopacket.LayerType) {
	if layerType != gopacket.LayerTypePayload {
		packet.Type = layerType.String()
	}
}

func unwrapVxlanPacket(pcapPacket gopacket.Packet, packet *model.Packet) gopacket.Packet {
	vxlan := pcapPacket.Layer(layers.LayerTypeVXLAN)
	if vxlan != nil {
		vxlan, _ := vxlan.(*layers.VXLAN)
		if vxlan.Payload != nil && len(vxlan.Payload) > 0 {
			oldData := pcapPacket.Metadata()
			pcapPacket = gopacket.NewPacket(vxlan.Payload, layers.LayerTypeEthernet, gopacket.Default)
			newData := pcapPacket.Metadata()
			newData.Timestamp = oldData.Timestamp
			newData.CaptureLength = len(vxlan.Payload)
			newData.Length = newData.CaptureLength
			if packet != nil {
				packet.Flags = append(packet.Flags, "VXLAN")
			}
		}
	}
	return pcapPacket
}

func parseData(pcapPacket gopacket.Packet, packet *model.Packet, unwrap bool) {
	if unwrap {
		pcapPacket = unwrapVxlanPacket(pcapPacket, packet)
	}

	packet.Timestamp = pcapPacket.Metadata().Timestamp
	packet.Length = pcapPacket.Metadata().Length

	layer := pcapPacket.Layer(layers.LayerTypeEthernet)
	if layer != nil {
		layer := layer.(*layers.Ethernet)
		packet.SrcMac = layer.SrcMAC.String()
		packet.DstMac = layer.DstMAC.String()
	}

	layer = pcapPacket.Layer(layers.LayerTypeIPv6)
	if layer != nil {
		layer := layer.(*layers.IPv6)
		packet.SrcIp = layer.SrcIP.String()
		packet.DstIp = layer.DstIP.String()
	} else {
		layer = pcapPacket.Layer(layers.LayerTypeIPv4)
		if layer != nil {
			layer := layer.(*layers.IPv4)
			packet.SrcIp = layer.SrcIP.String()
			packet.DstIp = layer.DstIP.String()
		}
	}

	for _, layerType := range SupportedLayerTypes {
		layer = pcapPacket.Layer(layerType)
		if layer != nil {
			overrideType(packet, layer.LayerType())
		}
	}

	layer = pcapPacket.Layer(layers.LayerTypeTCP)
	if layer != nil {
		layer := layer.(*layers.TCP)
		packet.SrcPort = int(layer.SrcPort)
		packet.DstPort = int(layer.DstPort)
		packet.Sequence = int(layer.Seq)
		packet.Acknowledge = int(layer.Ack)
		packet.Window = int(layer.Window)
		packet.Checksum = int(layer.Checksum)
		if layer.SYN {
			packet.Flags = append(packet.Flags, "SYN")
		}
		if layer.PSH {
			packet.Flags = append(packet.Flags, "PSH")
		}
		if layer.FIN {
			packet.Flags = append(packet.Flags, "FIN")
		}
		if layer.RST {
			packet.Flags = append(packet.Flags, "RST")
		}
		if layer.ACK {
			packet.Flags = append(packet.Flags, "ACK")
		}
		overrideType(packet, layer.SrcPort.LayerType())
		overrideType(packet, layer.DstPort.LayerType())
		overrideType(packet, layer.LayerType())
	}

	layer = pcapPacket.Layer(layers.LayerTypeUDP)
	if layer != nil {
		layer := layer.(*layers.UDP)
		packet.SrcPort = int(layer.SrcPort)
		packet.DstPort = int(layer.DstPort)
		packet.Checksum = int(layer.Checksum)
		overrideType(packet, layer.NextLayerType())
		overrideType(packet, layer.LayerType())
	}

	packetLayers := pcapPacket.Layers()
	topLayer := packetLayers[len(packetLayers)-1]
	overrideType(packet, topLayer.LayerType())

	packet.Payload = base64.StdEncoding.EncodeToString(pcapPacket.Data())
	packet.PayloadOffset = 0
	appLayer := pcapPacket.ApplicationLayer()
	if appLayer != nil {
		packet.PayloadOffset = len(pcapPacket.Data()) - len(appLayer.Payload())
	}
}
