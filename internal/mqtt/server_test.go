package mqtt

import (
	"testing"

	"mvp-platform/internal/model"
)

func TestResolveRouting(t *testing.T) {
	t.Parallel()

	routes := resolveRouting(model.ProductAccessProfile{Topic: "mvp/{device_id}/up"}, "dev-1")
	if routes.UpTopic != "mvp/dev-1/up" {
		t.Fatalf("UpTopic = %q, want %q", routes.UpTopic, "mvp/dev-1/up")
	}
	if routes.DownTopic != "mvp/dev-1/down" {
		t.Fatalf("DownTopic = %q, want %q", routes.DownTopic, "mvp/dev-1/down")
	}
	if routes.AckTopic != "mvp/dev-1/ack" {
		t.Fatalf("AckTopic = %q, want %q", routes.AckTopic, "mvp/dev-1/ack")
	}
}

func TestResolveRoutingFallsBackForWildcardTemplate(t *testing.T) {
	t.Parallel()

	routes := resolveRouting(model.ProductAccessProfile{Topic: "factory/+/+/up"}, "dev-2")
	if routes.UpTopic != "devices/dev-2/up" {
		t.Fatalf("UpTopic = %q, want fallback %q", routes.UpTopic, "devices/dev-2/up")
	}
}

func TestMQTTTopicMatch(t *testing.T) {
	t.Parallel()

	if !mqttTopicMatch("mvp/dev-1/down", "mvp/dev-1/down") {
		t.Fatal("expected exact match")
	}
	if !mqttTopicMatch("mvp/+/down", "mvp/dev-1/down") {
		t.Fatal("expected single-level wildcard match")
	}
	if !mqttTopicMatch("mvp/#", "mvp/dev-1/down") {
		t.Fatal("expected multi-level wildcard match")
	}
	if mqttTopicMatch("mvp/+/ack", "mvp/dev-1/down") {
		t.Fatal("unexpected wildcard match")
	}
}
