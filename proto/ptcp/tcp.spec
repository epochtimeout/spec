

enum MessageCode {
    UNDEFINED = 0;
    OPEN_STREAM = 1;
    CLOSE_STREAM = 2;
    STREAM_MESSAGE = 3;
}

message Message {
    code    MessageCode     1;
    open    OpenStream      2;
    close   CloseStream     3;
    message StreamMessage   4;
}

message OpenStream {
    id  bin128  1;
}

message CloseStream {
    id  bin128  1;
}

message StreamMessage {
    id      bin128  1;
    bytes   bytes   2;
}
