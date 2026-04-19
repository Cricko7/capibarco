abstract interface class ChatRealtimeSocket {
  Stream<dynamic> get messages;

  void sendText(String payload);

  Future<void> close();
}
