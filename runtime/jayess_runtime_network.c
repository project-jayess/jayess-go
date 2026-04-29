/* Split from jayess_runtime.c: network, TLS, and HTTP implementation cluster. */

static jayess_socket_handle jayess_std_socket_handle(jayess_value *env) {
    if (env == NULL || env->kind != JAYESS_VALUE_OBJECT || env->as.object_value == NULL) {
        return JAYESS_INVALID_SOCKET;
    }
    return env->as.object_value->socket_handle;
}

static void jayess_std_socket_set_handle(jayess_value *env, jayess_socket_handle handle) {
    if (env != NULL && env->kind == JAYESS_VALUE_OBJECT && env->as.object_value != NULL) {
        env->as.object_value->socket_handle = handle;
    }
}

static jayess_tls_socket_state *jayess_std_tls_state(jayess_value *env) {
    if (env == NULL || env->kind != JAYESS_VALUE_OBJECT || env->as.object_value == NULL) {
        return NULL;
    }
    if (!jayess_std_kind_is(env, "Socket")) {
        return NULL;
    }
    return (jayess_tls_socket_state *)env->as.object_value->native_handle;
}

static jayess_value *jayess_std_tls_peer_certificate(jayess_value *env) {
    jayess_tls_socket_state *state = jayess_std_tls_state(env);
    if (state == NULL) {
        return jayess_value_undefined();
    }
#ifdef _WIN32
    {
        PCCERT_CONTEXT cert = NULL;
        char subject[512];
        char issuer[512];
        char subject_cn[256];
        char issuer_cn[256];
        char serial[256];
        char valid_from[64];
        char valid_to[64];
        jayess_object *result;
        SECURITY_STATUS status = QueryContextAttributes(&state->context, SECPKG_ATTR_REMOTE_CERT_CONTEXT, &cert);
        if (status != SEC_E_OK || cert == NULL) {
            return jayess_value_undefined();
        }
        subject[0] = '\0';
        issuer[0] = '\0';
        subject_cn[0] = '\0';
        issuer_cn[0] = '\0';
        serial[0] = '\0';
        valid_from[0] = '\0';
        valid_to[0] = '\0';
        CertGetNameStringA(cert, CERT_NAME_SIMPLE_DISPLAY_TYPE, 0, NULL, subject, (DWORD)sizeof(subject));
        CertGetNameStringA(cert, CERT_NAME_SIMPLE_DISPLAY_TYPE, CERT_NAME_ISSUER_FLAG, NULL, issuer, (DWORD)sizeof(issuer));
        CertGetNameStringA(cert, CERT_NAME_ATTR_TYPE, 0, szOID_COMMON_NAME, subject_cn, (DWORD)sizeof(subject_cn));
        CertGetNameStringA(cert, CERT_NAME_ATTR_TYPE, CERT_NAME_ISSUER_FLAG, szOID_COMMON_NAME, issuer_cn, (DWORD)sizeof(issuer_cn));
        {
            int i;
            size_t offset = 0;
            for (i = (int)cert->pCertInfo->SerialNumber.cbData - 1; i >= 0 && offset + 2 < sizeof(serial); i--) {
                offset += (size_t)snprintf(serial + offset, sizeof(serial) - offset, "%02X", cert->pCertInfo->SerialNumber.pbData[i]);
            }
        }
        {
            SYSTEMTIME from_system;
            SYSTEMTIME to_system;
            if (FileTimeToSystemTime(&cert->pCertInfo->NotBefore, &from_system)) {
                snprintf(valid_from, sizeof(valid_from), "%04u-%02u-%02uT%02u:%02u:%02uZ",
                    (unsigned int)from_system.wYear, (unsigned int)from_system.wMonth, (unsigned int)from_system.wDay,
                    (unsigned int)from_system.wHour, (unsigned int)from_system.wMinute, (unsigned int)from_system.wSecond);
            }
            if (FileTimeToSystemTime(&cert->pCertInfo->NotAfter, &to_system)) {
                snprintf(valid_to, sizeof(valid_to), "%04u-%02u-%02uT%02u:%02u:%02uZ",
                    (unsigned int)to_system.wYear, (unsigned int)to_system.wMonth, (unsigned int)to_system.wDay,
                    (unsigned int)to_system.wHour, (unsigned int)to_system.wMinute, (unsigned int)to_system.wSecond);
            }
        }
        result = jayess_object_new();
        if (result == NULL) {
            CertFreeCertificateContext(cert);
            return jayess_value_from_object(NULL);
        }
        jayess_object_set_value(result, "subject", jayess_value_from_string(subject));
        jayess_object_set_value(result, "issuer", jayess_value_from_string(issuer));
        jayess_object_set_value(result, "subjectCN", jayess_value_from_string(subject_cn));
        jayess_object_set_value(result, "issuerCN", jayess_value_from_string(issuer_cn));
        jayess_object_set_value(result, "serialNumber", jayess_value_from_string(serial));
        jayess_object_set_value(result, "validFrom", jayess_value_from_string(valid_from));
        jayess_object_set_value(result, "validTo", jayess_value_from_string(valid_to));
        jayess_object_set_value(result, "subjectAltNames", jayess_std_tls_subject_alt_names(env));
        jayess_object_set_value(result, "backend", jayess_value_from_string("schannel"));
        jayess_object_set_value(result, "authorized", jayess_object_get(env->as.object_value, "authorized"));
        CertFreeCertificateContext(cert);
        return jayess_value_from_object(result);
    }
#else
    {
        X509 *cert = SSL_get_peer_certificate(state->ssl);
        char subject[512];
        char issuer[512];
        char subject_cn[256];
        char issuer_cn[256];
        char serial[256];
        char valid_from[64];
        char valid_to[64];
        jayess_object *result;
        if (cert == NULL) {
            return jayess_value_undefined();
        }
        subject[0] = '\0';
        issuer[0] = '\0';
        subject_cn[0] = '\0';
        issuer_cn[0] = '\0';
        serial[0] = '\0';
        valid_from[0] = '\0';
        valid_to[0] = '\0';
        X509_NAME_oneline(X509_get_subject_name(cert), subject, (int)sizeof(subject));
        X509_NAME_oneline(X509_get_issuer_name(cert), issuer, (int)sizeof(issuer));
        X509_NAME_get_text_by_NID(X509_get_subject_name(cert), NID_commonName, subject_cn, (int)sizeof(subject_cn));
        X509_NAME_get_text_by_NID(X509_get_issuer_name(cert), NID_commonName, issuer_cn, (int)sizeof(issuer_cn));
        {
            ASN1_INTEGER *serial_number = X509_get_serialNumber(cert);
            BIGNUM *bn = ASN1_INTEGER_to_BN(serial_number, NULL);
            if (bn != NULL) {
                char *hex = BN_bn2hex(bn);
                if (hex != NULL) {
                    snprintf(serial, sizeof(serial), "%s", hex);
                    OPENSSL_free(hex);
                }
                BN_free(bn);
            }
        }
        {
            const ASN1_TIME *not_before = X509_get0_notBefore(cert);
            const ASN1_TIME *not_after = X509_get0_notAfter(cert);
            BIO *bio = BIO_new(BIO_s_mem());
            if (bio != NULL) {
                if (not_before != NULL && ASN1_TIME_print(bio, not_before)) {
                    int len = BIO_read(bio, valid_from, (int)sizeof(valid_from) - 1);
                    if (len > 0) {
                        valid_from[len] = '\0';
                    }
                }
                (void)BIO_reset(bio);
                if (not_after != NULL && ASN1_TIME_print(bio, not_after)) {
                    int len = BIO_read(bio, valid_to, (int)sizeof(valid_to) - 1);
                    if (len > 0) {
                        valid_to[len] = '\0';
                    }
                }
                BIO_free(bio);
            }
        }
        result = jayess_object_new();
        if (result == NULL) {
            X509_free(cert);
            return jayess_value_from_object(NULL);
        }
        jayess_object_set_value(result, "subject", jayess_value_from_string(subject));
        jayess_object_set_value(result, "issuer", jayess_value_from_string(issuer));
        jayess_object_set_value(result, "subjectCN", jayess_value_from_string(subject_cn));
        jayess_object_set_value(result, "issuerCN", jayess_value_from_string(issuer_cn));
        jayess_object_set_value(result, "serialNumber", jayess_value_from_string(serial));
        jayess_object_set_value(result, "validFrom", jayess_value_from_string(valid_from));
        jayess_object_set_value(result, "validTo", jayess_value_from_string(valid_to));
        jayess_object_set_value(result, "subjectAltNames", jayess_std_tls_subject_alt_names(env));
        jayess_object_set_value(result, "backend", jayess_value_from_string("openssl"));
        jayess_object_set_value(result, "authorized", jayess_object_get(env->as.object_value, "authorized"));
        X509_free(cert);
        return jayess_value_from_object(result);
    }
#endif
}

static int jayess_std_tls_send_all(jayess_socket_handle handle, const unsigned char *buffer, size_t length) {
    size_t offset = 0;
    while (offset < length) {
        int sent = (int)send(handle, (const char *)buffer + offset, (int)(length - offset), 0);
        if (sent <= 0) {
            return 0;
        }
        offset += (size_t)sent;
    }
    return 1;
}

static int jayess_std_tls_state_free(jayess_tls_socket_state *state, int close_handle) {
    if (state == NULL) {
        return 1;
    }
#ifdef _WIN32
    if (state->has_context) {
        DeleteSecurityContext(&state->context);
        state->has_context = 0;
    }
    if (state->has_credentials) {
        FreeCredentialHandle(&state->credentials);
        state->has_credentials = 0;
    }
    free(state->encrypted_buffer);
    free(state->plaintext_buffer);
    free(state->host);
    state->encrypted_buffer = NULL;
    state->plaintext_buffer = NULL;
    state->host = NULL;
    state->encrypted_length = 0;
    state->encrypted_capacity = 0;
    state->plaintext_offset = 0;
    state->plaintext_length = 0;
#else
    if (state->ssl != NULL) {
        SSL_free(state->ssl);
        state->ssl = NULL;
    }
    if (state->ctx != NULL) {
        SSL_CTX_free(state->ctx);
        state->ctx = NULL;
    }
    free(state->host);
    state->host = NULL;
#endif
    if (close_handle && state->handle != JAYESS_INVALID_SOCKET) {
        jayess_std_socket_close_handle(state->handle);
        state->handle = JAYESS_INVALID_SOCKET;
    }
    free(state);
    return 1;
}

static int jayess_std_tls_read_bytes(jayess_value *env, unsigned char *buffer, int max_count, int *did_timeout) {
    jayess_tls_socket_state *state = jayess_std_tls_state(env);
    if (did_timeout != NULL) {
        *did_timeout = 0;
    }
    if (state == NULL || buffer == NULL || max_count <= 0) {
        return -1;
    }
#ifdef _WIN32
    while (1) {
        if (state->plaintext_offset < state->plaintext_length) {
            size_t available = state->plaintext_length - state->plaintext_offset;
            size_t count = available < (size_t)max_count ? available : (size_t)max_count;
            memcpy(buffer, state->plaintext_buffer + state->plaintext_offset, count);
            state->plaintext_offset += count;
            if (state->plaintext_offset >= state->plaintext_length) {
                state->plaintext_offset = 0;
                state->plaintext_length = 0;
            }
            return (int)count;
        }
        {
            SecBuffer buffers[4];
            SecBufferDesc descriptor;
            SECURITY_STATUS status;
            int i;
            if (state->encrypted_length == 0) {
                if (state->encrypted_capacity < 16384) {
                    unsigned char *grown = (unsigned char *)realloc(state->encrypted_buffer, 16384);
                    if (grown == NULL) {
                        return -1;
                    }
                    state->encrypted_buffer = grown;
                    state->encrypted_capacity = 16384;
                }
                {
                    int read_count = (int)recv(state->handle, (char *)state->encrypted_buffer, (int)state->encrypted_capacity, 0);
                    if (read_count == 0) {
                        return 0;
                    }
                    if (read_count < 0) {
                        int error_code = WSAGetLastError();
                        if (did_timeout != NULL && error_code == WSAETIMEDOUT) {
                            *did_timeout = 1;
                        }
                        return -1;
                    }
                    state->encrypted_length = (size_t)read_count;
                }
            }
            buffers[0].pvBuffer = state->encrypted_buffer;
            buffers[0].cbBuffer = (unsigned long)state->encrypted_length;
            buffers[0].BufferType = SECBUFFER_DATA;
            buffers[1].pvBuffer = NULL;
            buffers[1].cbBuffer = 0;
            buffers[1].BufferType = SECBUFFER_EMPTY;
            buffers[2].pvBuffer = NULL;
            buffers[2].cbBuffer = 0;
            buffers[2].BufferType = SECBUFFER_EMPTY;
            buffers[3].pvBuffer = NULL;
            buffers[3].cbBuffer = 0;
            buffers[3].BufferType = SECBUFFER_EMPTY;
            descriptor.ulVersion = SECBUFFER_VERSION;
            descriptor.cBuffers = 4;
            descriptor.pBuffers = buffers;
            status = DecryptMessage(&state->context, &descriptor, 0, NULL);
            if (status == SEC_E_INCOMPLETE_MESSAGE) {
                if (state->encrypted_length >= state->encrypted_capacity) {
                    size_t new_capacity = state->encrypted_capacity > 0 ? state->encrypted_capacity * 2 : 32768;
                    unsigned char *grown = (unsigned char *)realloc(state->encrypted_buffer, new_capacity);
                    if (grown == NULL) {
                        return -1;
                    }
                    state->encrypted_buffer = grown;
                    state->encrypted_capacity = new_capacity;
                }
                {
                    int read_count = (int)recv(state->handle, (char *)state->encrypted_buffer + state->encrypted_length, (int)(state->encrypted_capacity - state->encrypted_length), 0);
                    if (read_count == 0) {
                        return 0;
                    }
                    if (read_count < 0) {
                        int error_code = WSAGetLastError();
                        if (did_timeout != NULL && error_code == WSAETIMEDOUT) {
                            *did_timeout = 1;
                        }
                        return -1;
                    }
                    state->encrypted_length += (size_t)read_count;
                }
                continue;
            }
            if (status == SEC_I_CONTEXT_EXPIRED) {
                return 0;
            }
            if (status != SEC_E_OK) {
                return -1;
            }
            for (i = 0; i < 4; i++) {
                if (buffers[i].BufferType == SECBUFFER_DATA && buffers[i].cbBuffer > 0) {
                    unsigned char *plain = (unsigned char *)buffers[i].pvBuffer;
                    unsigned long plain_len = buffers[i].cbBuffer;
                    if (state->plaintext_buffer == NULL || state->plaintext_length < plain_len) {
                        unsigned char *grown = (unsigned char *)realloc(state->plaintext_buffer, (size_t)plain_len);
                        if (grown == NULL) {
                            return -1;
                        }
                        state->plaintext_buffer = grown;
                    }
                    memcpy(state->plaintext_buffer, plain, plain_len);
                    state->plaintext_offset = 0;
                    state->plaintext_length = plain_len;
                    break;
                }
            }
            for (i = 0; i < 4; i++) {
                if (buffers[i].BufferType == SECBUFFER_EXTRA) {
                    memmove(state->encrypted_buffer, state->encrypted_buffer + (state->encrypted_length - buffers[i].cbBuffer), buffers[i].cbBuffer);
                    state->encrypted_length = buffers[i].cbBuffer;
                    break;
                }
            }
            if (i == 4) {
                state->encrypted_length = 0;
            }
        }
    }
#else
    {
        int read_count = SSL_read(state->ssl, buffer, max_count);
        if (read_count > 0) {
            return read_count;
        }
        {
            int ssl_error = SSL_get_error(state->ssl, read_count);
            if (ssl_error == SSL_ERROR_ZERO_RETURN) {
                return 0;
            }
            if (ssl_error == SSL_ERROR_WANT_READ || ssl_error == SSL_ERROR_WANT_WRITE || (ssl_error == SSL_ERROR_SYSCALL && (errno == EAGAIN || errno == EWOULDBLOCK))) {
                if (did_timeout != NULL) {
                    *did_timeout = 1;
                }
            }
            return -1;
        }
    }
#endif
}

static int jayess_std_tls_write_bytes(jayess_value *env, const unsigned char *buffer, int length, int *did_timeout) {
    jayess_tls_socket_state *state = jayess_std_tls_state(env);
    int offset = 0;
    if (did_timeout != NULL) {
        *did_timeout = 0;
    }
    if (state == NULL || buffer == NULL || length < 0) {
        return -1;
    }
#ifdef _WIN32
    while (offset < length) {
        int chunk_size = length - offset;
        int total_size;
        unsigned char *packet;
        SecBuffer buffers[4];
        SecBufferDesc descriptor;
        SECURITY_STATUS status;
        if (chunk_size > (int)state->stream_sizes.cbMaximumMessage) {
            chunk_size = (int)state->stream_sizes.cbMaximumMessage;
        }
        total_size = (int)(state->stream_sizes.cbHeader + chunk_size + state->stream_sizes.cbTrailer);
        packet = (unsigned char *)malloc((size_t)total_size);
        if (packet == NULL) {
            return -1;
        }
        memcpy(packet + state->stream_sizes.cbHeader, buffer + offset, (size_t)chunk_size);
        buffers[0].pvBuffer = packet;
        buffers[0].cbBuffer = state->stream_sizes.cbHeader;
        buffers[0].BufferType = SECBUFFER_STREAM_HEADER;
        buffers[1].pvBuffer = packet + state->stream_sizes.cbHeader;
        buffers[1].cbBuffer = (unsigned long)chunk_size;
        buffers[1].BufferType = SECBUFFER_DATA;
        buffers[2].pvBuffer = packet + state->stream_sizes.cbHeader + chunk_size;
        buffers[2].cbBuffer = state->stream_sizes.cbTrailer;
        buffers[2].BufferType = SECBUFFER_STREAM_TRAILER;
        buffers[3].pvBuffer = NULL;
        buffers[3].cbBuffer = 0;
        buffers[3].BufferType = SECBUFFER_EMPTY;
        descriptor.ulVersion = SECBUFFER_VERSION;
        descriptor.cBuffers = 4;
        descriptor.pBuffers = buffers;
        status = EncryptMessage(&state->context, 0, &descriptor, 0);
        if (status != SEC_E_OK) {
            free(packet);
            return -1;
        }
        if (!jayess_std_tls_send_all(state->handle, packet, buffers[0].cbBuffer + buffers[1].cbBuffer + buffers[2].cbBuffer)) {
            int error_code = WSAGetLastError();
            if (did_timeout != NULL && error_code == WSAETIMEDOUT) {
                *did_timeout = 1;
            }
            free(packet);
            return -1;
        }
        free(packet);
        offset += chunk_size;
    }
    return length;
#else
    while (offset < length) {
        int written = SSL_write(state->ssl, buffer + offset, length - offset);
        if (written > 0) {
            offset += written;
            continue;
        }
        {
            int ssl_error = SSL_get_error(state->ssl, written);
            if (ssl_error == SSL_ERROR_WANT_READ || ssl_error == SSL_ERROR_WANT_WRITE || (ssl_error == SSL_ERROR_SYSCALL && (errno == EAGAIN || errno == EWOULDBLOCK))) {
                if (did_timeout != NULL) {
                    *did_timeout = 1;
                }
            }
            return -1;
        }
    }
    return length;
#endif
}

static jayess_value *jayess_std_tls_connect_socket(jayess_value *options) {
    jayess_object *object_options = jayess_value_as_object(options);
    jayess_value *host_value = object_options != NULL ? jayess_object_get(object_options, "host") : NULL;
    jayess_value *port_value = object_options != NULL ? jayess_object_get(object_options, "port") : NULL;
    jayess_value *reject_value = object_options != NULL ? jayess_object_get(object_options, "rejectUnauthorized") : NULL;
    jayess_value *timeout_value = object_options != NULL ? jayess_object_get(object_options, "timeout") : NULL;
    jayess_value *alpn_value = object_options != NULL ? jayess_object_get(object_options, "alpnProtocols") : NULL;
    jayess_value *server_name_value = object_options != NULL ? jayess_object_get(object_options, "serverName") : NULL;
    jayess_value *ca_file_value = object_options != NULL ? jayess_object_get(object_options, "caFile") : NULL;
    jayess_value *ca_path_value = object_options != NULL ? jayess_object_get(object_options, "caPath") : NULL;
    jayess_value *trust_system_value = object_options != NULL ? jayess_object_get(object_options, "trustSystem") : NULL;
    char *host_text = jayess_value_stringify(host_value);
    char *server_name_text = NULL;
    char *ca_file_text = NULL;
    char *ca_path_text = NULL;
    int port = (int)jayess_value_to_number(port_value);
    int reject_unauthorized = reject_value == NULL || reject_value->kind == JAYESS_VALUE_UNDEFINED ? 1 : jayess_value_as_bool(reject_value);
    int timeout = (int)jayess_value_to_number(timeout_value);
    int trust_system = trust_system_value == NULL || trust_system_value->kind == JAYESS_VALUE_UNDEFINED ? 1 : jayess_value_as_bool(trust_system_value);
    jayess_value *normalized_alpn = jayess_value_undefined();
    unsigned char *alpn_wire = NULL;
    size_t alpn_wire_length = 0;
    char negotiated_alpn[256];
    const char *negotiated_protocol = "";
    char port_text[32];
    struct addrinfo hints;
    struct addrinfo *results = NULL;
    struct addrinfo *entry;
    jayess_socket_handle handle = JAYESS_INVALID_SOCKET;
    int family = 0;
    int status;
    jayess_tls_socket_state *state = NULL;
#ifdef _WIN32
    SCHANNEL_CRED credentials;
    TimeStamp expiry;
    DWORD flags = ISC_REQ_SEQUENCE_DETECT | ISC_REQ_REPLAY_DETECT | ISC_REQ_CONFIDENTIALITY |
        ISC_REQ_EXTENDED_ERROR | ISC_REQ_ALLOCATE_MEMORY | ISC_REQ_STREAM;
    SecBuffer out_buffer;
    SecBufferDesc out_desc;
    SecBuffer in_buffers[2];
    SecBufferDesc in_desc;
    SecBuffer initial_in_buffers[1];
    SecBufferDesc initial_in_desc;
    DWORD context_flags = 0;
    SECURITY_STATUS sec_status;
    int first_call = 1;
    void *alpn_buffer = NULL;
    unsigned long alpn_buffer_length = 0;
#else
    int authorized = 0;
#endif
    negotiated_alpn[0] = '\0';
    if (host_text == NULL || host_text[0] == '\0' || port <= 0 || !jayess_std_socket_runtime_ready()) {
        free(host_text);
        return jayess_value_undefined();
    }
    if (server_name_value != NULL && server_name_value->kind != JAYESS_VALUE_UNDEFINED && server_name_value->kind != JAYESS_VALUE_NULL) {
        server_name_text = jayess_value_stringify(server_name_value);
    } else {
        server_name_text = jayess_strdup(host_text);
    }
    if (ca_file_value != NULL && ca_file_value->kind != JAYESS_VALUE_UNDEFINED && ca_file_value->kind != JAYESS_VALUE_NULL) {
        ca_file_text = jayess_value_stringify(ca_file_value);
    }
    if (ca_path_value != NULL && ca_path_value->kind != JAYESS_VALUE_UNDEFINED && ca_path_value->kind != JAYESS_VALUE_NULL) {
        ca_path_text = jayess_value_stringify(ca_path_value);
    }
    if (server_name_text == NULL || server_name_text[0] == '\0') {
        free(host_text);
        free(server_name_text);
        free(ca_file_text);
        free(ca_path_text);
        jayess_throw(jayess_type_error_value("tls.connect serverName must be a non-empty string"));
        return jayess_value_undefined();
    }
    if (alpn_value != NULL && alpn_value->kind != JAYESS_VALUE_UNDEFINED && alpn_value->kind != JAYESS_VALUE_NULL) {
        normalized_alpn = jayess_std_tls_normalize_alpn_protocols(alpn_value);
        if (jayess_has_exception()) {
            free(host_text);
            return jayess_value_undefined();
        }
    }
    if (normalized_alpn != NULL && normalized_alpn->kind == JAYESS_VALUE_ARRAY && normalized_alpn->as.array_value != NULL && normalized_alpn->as.array_value->count > 0) {
        if (!jayess_std_tls_build_alpn_wire(normalized_alpn, &alpn_wire, &alpn_wire_length)) {
            jayess_throw(jayess_type_error_value("tls.connect failed to encode ALPN protocols"));
            free(host_text);
            return jayess_value_undefined();
        }
    }
    snprintf(port_text, sizeof(port_text), "%d", port);
    memset(&hints, 0, sizeof(hints));
    hints.ai_family = AF_UNSPEC;
    hints.ai_socktype = SOCK_STREAM;
    status = getaddrinfo(host_text, port_text, &hints, &results);
    if (status != 0 || results == NULL) {
        jayess_throw(jayess_type_error_value("tls.connect failed to resolve host"));
        free(host_text);
        return jayess_value_undefined();
    }
    for (entry = results; entry != NULL; entry = entry->ai_next) {
        handle = socket(entry->ai_family, entry->ai_socktype, entry->ai_protocol);
        if (handle == JAYESS_INVALID_SOCKET) {
            continue;
        }
        if (connect(handle, entry->ai_addr, (int)entry->ai_addrlen) == 0) {
            family = entry->ai_family == AF_INET6 ? 6 : 4;
            break;
        }
        jayess_std_socket_close_handle(handle);
        handle = JAYESS_INVALID_SOCKET;
    }
    freeaddrinfo(results);
    if (handle == JAYESS_INVALID_SOCKET) {
        jayess_throw(jayess_type_error_value("tls.connect failed to connect socket"));
        free(host_text);
        return jayess_value_undefined();
    }
    if (timeout > 0 && !jayess_std_socket_configure_timeout(handle, timeout)) {
        jayess_std_socket_close_handle(handle);
        jayess_throw(jayess_type_error_value("tls.connect failed to configure timeout"));
        free(host_text);
        return jayess_value_undefined();
    }
    state = (jayess_tls_socket_state *)calloc(1, sizeof(jayess_tls_socket_state));
    if (state == NULL) {
        jayess_std_socket_close_handle(handle);
        jayess_throw(jayess_type_error_value("tls.connect failed to allocate TLS state"));
        free(host_text);
        return jayess_value_undefined();
    }
    state->handle = handle;
    state->reject_unauthorized = reject_unauthorized;
    state->host = jayess_strdup(server_name_text);
#ifdef _WIN32
    int custom_trust_requested = ((ca_file_text != NULL && ca_file_text[0] != '\0') || (ca_path_text != NULL && ca_path_text[0] != '\0') || !trust_system);
    if ((ca_file_text != NULL && ca_file_text[0] != '\0') || (ca_path_text != NULL && ca_path_text[0] != '\0') || !trust_system) {
        if (!reject_unauthorized) {
            /* Custom trust settings are ignored when certificate verification is disabled. */
            custom_trust_requested = 0;
        }
    }
    if (alpn_wire != NULL && alpn_wire_length > 0) {
        alpn_buffer = jayess_std_tls_build_schannel_alpn_buffer(alpn_wire, alpn_wire_length, &alpn_buffer_length);
        if (alpn_buffer == NULL) {
            free(alpn_wire);
            jayess_std_tls_state_free(state, 1);
            jayess_throw(jayess_type_error_value("tls.connect failed to prepare ALPN protocols"));
            free(host_text);
            free(server_name_text);
            free(ca_file_text);
            free(ca_path_text);
            return jayess_value_undefined();
        }
    }
    memset(&credentials, 0, sizeof(credentials));
    credentials.dwVersion = SCHANNEL_CRED_VERSION;
    credentials.dwFlags = SCH_USE_STRONG_CRYPTO | SCH_CRED_NO_DEFAULT_CREDS |
        ((reject_unauthorized && !custom_trust_requested) ? SCH_CRED_AUTO_CRED_VALIDATION : SCH_CRED_MANUAL_CRED_VALIDATION);
    sec_status = AcquireCredentialsHandleA(NULL, UNISP_NAME_A, SECPKG_CRED_OUTBOUND, NULL, &credentials, NULL, NULL, &state->credentials, &expiry);
    if (sec_status != SEC_E_OK) {
        jayess_std_tls_state_free(state, 1);
        jayess_throw(jayess_type_error_value("tls.connect failed to acquire TLS credentials"));
        free(host_text);
        free(server_name_text);
        free(ca_file_text);
        free(ca_path_text);
        return jayess_value_undefined();
    }
    state->has_credentials = 1;
    while (1) {
        out_buffer.pvBuffer = NULL;
        out_buffer.cbBuffer = 0;
        out_buffer.BufferType = SECBUFFER_TOKEN;
        out_desc.ulVersion = SECBUFFER_VERSION;
        out_desc.cBuffers = 1;
        out_desc.pBuffers = &out_buffer;
        if (first_call) {
            if (alpn_buffer != NULL) {
                initial_in_buffers[0].pvBuffer = alpn_buffer;
                initial_in_buffers[0].cbBuffer = alpn_buffer_length;
                initial_in_buffers[0].BufferType = SECBUFFER_APPLICATION_PROTOCOLS;
                initial_in_desc.ulVersion = SECBUFFER_VERSION;
                initial_in_desc.cBuffers = 1;
                initial_in_desc.pBuffers = initial_in_buffers;
                sec_status = InitializeSecurityContextA(&state->credentials, NULL, state->host, flags, 0, SECURITY_NATIVE_DREP, &initial_in_desc, 0, &state->context, &out_desc, &context_flags, &expiry);
            } else {
                sec_status = InitializeSecurityContextA(&state->credentials, NULL, state->host, flags, 0, SECURITY_NATIVE_DREP, NULL, 0, &state->context, &out_desc, &context_flags, &expiry);
            }
        } else {
            in_buffers[0].pvBuffer = state->encrypted_buffer;
            in_buffers[0].cbBuffer = (unsigned long)state->encrypted_length;
            in_buffers[0].BufferType = SECBUFFER_TOKEN;
            in_buffers[1].pvBuffer = NULL;
            in_buffers[1].cbBuffer = 0;
            in_buffers[1].BufferType = SECBUFFER_EMPTY;
            in_desc.ulVersion = SECBUFFER_VERSION;
            in_desc.cBuffers = 2;
            in_desc.pBuffers = in_buffers;
            sec_status = InitializeSecurityContextA(&state->credentials, &state->context, state->host, flags, 0, SECURITY_NATIVE_DREP, &in_desc, 0, &state->context, &out_desc, &context_flags, &expiry);
        }
        if (sec_status == SEC_E_OK || sec_status == SEC_I_CONTINUE_NEEDED || sec_status == SEC_I_COMPLETE_NEEDED || sec_status == SEC_I_COMPLETE_AND_CONTINUE || sec_status == SEC_E_INCOMPLETE_MESSAGE) {
            state->has_context = 1;
        }
        if (sec_status == SEC_I_COMPLETE_NEEDED || sec_status == SEC_I_COMPLETE_AND_CONTINUE) {
            if (CompleteAuthToken(&state->context, &out_desc) != SEC_E_OK) {
                if (out_buffer.pvBuffer != NULL) {
                    FreeContextBuffer(out_buffer.pvBuffer);
                }
                jayess_std_tls_state_free(state, 1);
                jayess_throw(jayess_type_error_value("tls.connect failed to complete TLS handshake token"));
                free(host_text);
                free(server_name_text);
                free(ca_file_text);
                free(ca_path_text);
                return jayess_value_undefined();
            }
        }
        if (out_buffer.pvBuffer != NULL && out_buffer.cbBuffer > 0) {
            int sent_ok = jayess_std_tls_send_all(handle, (const unsigned char *)out_buffer.pvBuffer, out_buffer.cbBuffer);
            FreeContextBuffer(out_buffer.pvBuffer);
            if (!sent_ok) {
                jayess_std_tls_state_free(state, 1);
                jayess_throw(jayess_type_error_value("tls.connect failed to send handshake bytes"));
                free(host_text);
                free(server_name_text);
                free(ca_file_text);
                free(ca_path_text);
                return jayess_value_undefined();
            }
        }
        if (sec_status == SEC_E_OK || sec_status == SEC_I_COMPLETE_NEEDED) {
            if (!first_call && in_buffers[1].BufferType == SECBUFFER_EXTRA && in_buffers[1].cbBuffer > 0) {
                memmove(state->encrypted_buffer, state->encrypted_buffer + (state->encrypted_length - in_buffers[1].cbBuffer), in_buffers[1].cbBuffer);
                state->encrypted_length = in_buffers[1].cbBuffer;
            } else {
                state->encrypted_length = 0;
            }
            break;
        }
        if (sec_status != SEC_I_CONTINUE_NEEDED && sec_status != SEC_I_COMPLETE_AND_CONTINUE && sec_status != SEC_E_INCOMPLETE_MESSAGE) {
            jayess_std_tls_state_free(state, 1);
            jayess_throw(jayess_type_error_value("tls.connect handshake failed"));
            free(host_text);
            free(server_name_text);
            free(ca_file_text);
            free(ca_path_text);
            return jayess_value_undefined();
        }
        if (!first_call) {
            if (in_buffers[1].BufferType == SECBUFFER_EXTRA && in_buffers[1].cbBuffer > 0 && in_buffers[1].cbBuffer < state->encrypted_length) {
                memmove(state->encrypted_buffer, state->encrypted_buffer + (state->encrypted_length - in_buffers[1].cbBuffer), in_buffers[1].cbBuffer);
                state->encrypted_length = in_buffers[1].cbBuffer;
            } else if (sec_status != SEC_E_INCOMPLETE_MESSAGE) {
                state->encrypted_length = 0;
            }
        }
        if (state->encrypted_capacity - state->encrypted_length < 4096) {
            size_t new_capacity = state->encrypted_capacity > 0 ? state->encrypted_capacity * 2 : 32768;
            unsigned char *grown = (unsigned char *)realloc(state->encrypted_buffer, new_capacity);
            if (grown == NULL) {
                jayess_std_tls_state_free(state, 1);
                jayess_throw(jayess_type_error_value("tls.connect failed to grow handshake buffer"));
                free(host_text);
                free(server_name_text);
                free(ca_file_text);
                free(ca_path_text);
                return jayess_value_undefined();
            }
            state->encrypted_buffer = grown;
            state->encrypted_capacity = new_capacity;
        }
        {
            int read_count = (int)recv(handle, (char *)state->encrypted_buffer + state->encrypted_length, (int)(state->encrypted_capacity - state->encrypted_length), 0);
            if (read_count <= 0) {
                jayess_std_tls_state_free(state, 1);
                jayess_throw(jayess_type_error_value("tls.connect failed while reading handshake bytes"));
                free(host_text);
                free(server_name_text);
                free(ca_file_text);
                free(ca_path_text);
                return jayess_value_undefined();
            }
            state->encrypted_length += (size_t)read_count;
        }
        first_call = 0;
        state->has_context = 1;
    }
    if (QueryContextAttributes(&state->context, SECPKG_ATTR_STREAM_SIZES, &state->stream_sizes) != SEC_E_OK) {
        free(alpn_buffer);
        free(alpn_wire);
        jayess_std_tls_state_free(state, 1);
        jayess_throw(jayess_type_error_value("tls.connect failed to query TLS stream sizes"));
        free(host_text);
        free(server_name_text);
        free(ca_file_text);
        free(ca_path_text);
        return jayess_value_undefined();
    }
    {
        SecPkgContext_ConnectionInfo connection_info;
        SecPkgContext_ApplicationProtocol application_protocol;
        int authorized = 0;
        if (QueryContextAttributes(&state->context, SECPKG_ATTR_CONNECTION_INFO, &connection_info) == SEC_E_OK) {
            negotiated_protocol = jayess_std_tls_windows_protocol_name(connection_info.dwProtocol);
        }
        if (QueryContextAttributes(&state->context, SECPKG_ATTR_APPLICATION_PROTOCOL, &application_protocol) == SEC_E_OK &&
            application_protocol.ProtoNegoStatus == SecApplicationProtocolNegotiationStatus_Success &&
            application_protocol.ProtocolIdSize > 0) {
            size_t copy_length = application_protocol.ProtocolIdSize;
            if (copy_length >= sizeof(negotiated_alpn)) {
                copy_length = sizeof(negotiated_alpn) - 1;
            }
            memcpy(negotiated_alpn, application_protocol.ProtocolId, copy_length);
            negotiated_alpn[copy_length] = '\0';
        }
        authorized = reject_unauthorized ? (!custom_trust_requested ? 1 : jayess_std_windows_validate_tls_certificate(state, server_name_text, ca_file_text, ca_path_text, trust_system)) : 0;
        if (reject_unauthorized && !authorized) {
            free(alpn_buffer);
            free(alpn_wire);
            jayess_std_tls_state_free(state, 1);
            jayess_throw(jayess_type_error_value("tls.connect certificate validation failed"));
            free(host_text);
            free(server_name_text);
            free(ca_file_text);
            free(ca_path_text);
            return jayess_value_undefined();
        }
        jayess_value *result = jayess_std_socket_value_from_handle(handle, host_text, port);
        if (result == NULL || result->kind != JAYESS_VALUE_OBJECT || result->as.object_value == NULL) {
            free(alpn_buffer);
            free(alpn_wire);
            jayess_std_tls_state_free(state, 1);
            jayess_throw(jayess_type_error_value("tls.connect failed to create socket object"));
            free(host_text);
            free(server_name_text);
            free(ca_file_text);
            free(ca_path_text);
            return jayess_value_undefined();
        }
        result->as.object_value->native_handle = state;
        jayess_object_set_value(result->as.object_value, "secure", jayess_value_from_bool(1));
        jayess_object_set_value(result->as.object_value, "authorized", jayess_value_from_bool(authorized));
        jayess_object_set_value(result->as.object_value, "backend", jayess_value_from_string("schannel"));
        jayess_object_set_value(result->as.object_value, "protocol", jayess_value_from_string(negotiated_protocol));
        jayess_object_set_value(result->as.object_value, "alpnProtocol", jayess_value_from_string(negotiated_alpn));
        jayess_object_set_value(result->as.object_value, "alpnProtocols", normalized_alpn != NULL && normalized_alpn->kind != JAYESS_VALUE_UNDEFINED ? normalized_alpn : jayess_value_from_array(jayess_array_new()));
        jayess_object_set_value(result->as.object_value, "timeout", jayess_value_from_number((double)timeout));
        jayess_std_socket_set_remote_family(result, family);
        jayess_std_socket_set_local_endpoint(result, handle);
        free(alpn_buffer);
        free(alpn_wire);
        free(host_text);
        free(server_name_text);
        free(ca_file_text);
        free(ca_path_text);
        return result;
    }
#else
    OPENSSL_init_ssl(0, NULL);
    state->ctx = SSL_CTX_new(TLS_client_method());
    if (state->ctx == NULL) {
        jayess_std_tls_state_free(state, 1);
        jayess_throw(jayess_type_error_value("tls.connect failed to create TLS context"));
        free(host_text);
        free(server_name_text);
        free(ca_file_text);
        free(ca_path_text);
        return jayess_value_undefined();
    }
    if (reject_unauthorized) {
        SSL_CTX_set_verify(state->ctx, SSL_VERIFY_PEER, NULL);
        if (trust_system) {
            SSL_CTX_set_default_verify_paths(state->ctx);
        }
        if ((ca_file_text != NULL && ca_file_text[0] != '\0') || (ca_path_text != NULL && ca_path_text[0] != '\0')) {
            if (SSL_CTX_load_verify_locations(state->ctx,
                    (ca_file_text != NULL && ca_file_text[0] != '\0') ? ca_file_text : NULL,
                    (ca_path_text != NULL && ca_path_text[0] != '\0') ? ca_path_text : NULL) != 1) {
                jayess_std_tls_state_free(state, 1);
                jayess_throw(jayess_type_error_value("tls.connect failed to load custom trust configuration"));
                free(host_text);
                free(server_name_text);
                free(ca_file_text);
                free(ca_path_text);
                return jayess_value_undefined();
            }
        } else if (!trust_system) {
            jayess_std_tls_state_free(state, 1);
            jayess_throw(jayess_type_error_value("tls.connect requires caFile or caPath when trustSystem is false"));
            free(host_text);
            free(server_name_text);
            free(ca_file_text);
            free(ca_path_text);
            return jayess_value_undefined();
        }
    } else {
        SSL_CTX_set_verify(state->ctx, SSL_VERIFY_NONE, NULL);
    }
    state->ssl = SSL_new(state->ctx);
    if (state->ssl == NULL) {
        jayess_std_tls_state_free(state, 1);
        jayess_throw(jayess_type_error_value("tls.connect failed to create TLS session"));
        free(host_text);
        free(server_name_text);
        free(ca_file_text);
        free(ca_path_text);
        return jayess_value_undefined();
    }
    SSL_set_fd(state->ssl, handle);
    SSL_set_tlsext_host_name(state->ssl, server_name_text);
    if (alpn_wire != NULL && alpn_wire_length > 0 && SSL_set_alpn_protos(state->ssl, alpn_wire, (unsigned int)alpn_wire_length) != 0) {
        free(alpn_wire);
        jayess_std_tls_state_free(state, 1);
        jayess_throw(jayess_type_error_value("tls.connect failed to configure ALPN protocols"));
        free(host_text);
        free(server_name_text);
        free(ca_file_text);
        free(ca_path_text);
        return jayess_value_undefined();
    }
    if (reject_unauthorized) {
        X509_VERIFY_PARAM *param = SSL_get0_param(state->ssl);
        if (param != NULL) {
            X509_VERIFY_PARAM_set1_host(param, server_name_text, 0);
        }
    }
    if (SSL_connect(state->ssl) != 1) {
        jayess_std_tls_state_free(state, 1);
        jayess_throw(jayess_type_error_value("tls.connect handshake failed"));
        free(host_text);
        free(server_name_text);
        free(ca_file_text);
        free(ca_path_text);
        return jayess_value_undefined();
    }
    negotiated_protocol = SSL_get_version(state->ssl);
    {
        const unsigned char *selected = NULL;
        unsigned int selected_length = 0;
        SSL_get0_alpn_selected(state->ssl, &selected, &selected_length);
        if (selected != NULL && selected_length > 0) {
            size_t copy_length = selected_length;
            if (copy_length >= sizeof(negotiated_alpn)) {
                copy_length = sizeof(negotiated_alpn) - 1;
            }
            memcpy(negotiated_alpn, selected, copy_length);
            negotiated_alpn[copy_length] = '\0';
        }
    }
    authorized = reject_unauthorized ? (SSL_get_verify_result(state->ssl) == X509_V_OK) : 0;
    {
        jayess_value *result = jayess_std_socket_value_from_handle(handle, host_text, port);
        if (result == NULL || result->kind != JAYESS_VALUE_OBJECT || result->as.object_value == NULL) {
            free(alpn_wire);
            jayess_std_tls_state_free(state, 1);
            jayess_throw(jayess_type_error_value("tls.connect failed to create socket object"));
            free(host_text);
            free(server_name_text);
            free(ca_file_text);
            free(ca_path_text);
            return jayess_value_undefined();
        }
        result->as.object_value->native_handle = state;
        jayess_object_set_value(result->as.object_value, "secure", jayess_value_from_bool(1));
        jayess_object_set_value(result->as.object_value, "authorized", jayess_value_from_bool(authorized));
        jayess_object_set_value(result->as.object_value, "backend", jayess_value_from_string("openssl"));
        jayess_object_set_value(result->as.object_value, "protocol", jayess_value_from_string(negotiated_protocol != NULL ? negotiated_protocol : ""));
        jayess_object_set_value(result->as.object_value, "alpnProtocol", jayess_value_from_string(negotiated_alpn));
        jayess_object_set_value(result->as.object_value, "alpnProtocols", normalized_alpn != NULL && normalized_alpn->kind != JAYESS_VALUE_UNDEFINED ? normalized_alpn : jayess_value_from_array(jayess_array_new()));
        jayess_object_set_value(result->as.object_value, "timeout", jayess_value_from_number((double)timeout));
        jayess_std_socket_set_remote_family(result, family);
        jayess_std_socket_set_local_endpoint(result, handle);
        free(alpn_wire);
        free(host_text);
        free(server_name_text);
        free(ca_file_text);
        free(ca_path_text);
        return result;
    }
#endif
}

static jayess_value *jayess_std_tls_accept_socket(jayess_value *socket_value, jayess_value *options) {
    jayess_object *object_options = jayess_value_as_object(options);
    jayess_value *cert_value = object_options != NULL ? jayess_object_get(object_options, "cert") : NULL;
    jayess_value *key_value = object_options != NULL ? jayess_object_get(object_options, "key") : NULL;
    jayess_value *cert_file_value = object_options != NULL ? jayess_object_get(object_options, "certFile") : NULL;
    jayess_value *key_file_value = object_options != NULL ? jayess_object_get(object_options, "keyFile") : NULL;
    jayess_socket_handle handle = jayess_std_socket_handle(socket_value);
    char *cert_text = NULL;
    char *key_text = NULL;
    jayess_tls_socket_state *state = NULL;
#ifdef _WIN32
    (void)socket_value;
    (void)object_options;
    (void)cert_value;
    (void)key_value;
    (void)cert_file_value;
    (void)key_file_value;
    (void)handle;
    jayess_throw(jayess_type_error_value("https.createServer is not available on this platform"));
    return jayess_value_undefined();
#else
    if (socket_value == NULL || socket_value->kind != JAYESS_VALUE_OBJECT || socket_value->as.object_value == NULL || !jayess_std_kind_is(socket_value, "Socket")) {
        jayess_throw(jayess_type_error_value("https.createServer accepted an invalid socket"));
        return jayess_value_undefined();
    }
    if (object_options == NULL) {
        jayess_throw(jayess_type_error_value("https.createServer options must be an object"));
        return jayess_value_undefined();
    }
    cert_text = jayess_value_stringify(!jayess_value_is_nullish(cert_value) ? cert_value : cert_file_value);
    key_text = jayess_value_stringify(!jayess_value_is_nullish(key_value) ? key_value : key_file_value);
    if (handle == JAYESS_INVALID_SOCKET) {
        free(cert_text);
        free(key_text);
        jayess_throw(jayess_type_error_value("https.createServer accepted an invalid socket handle"));
        return jayess_value_undefined();
    }
    if (cert_text == NULL || cert_text[0] == '\0' || key_text == NULL || key_text[0] == '\0') {
        free(cert_text);
        free(key_text);
        jayess_throw(jayess_type_error_value("https.createServer requires cert and key file paths"));
        return jayess_value_undefined();
    }
    OPENSSL_init_ssl(0, NULL);
    state = (jayess_tls_socket_state *)calloc(1, sizeof(jayess_tls_socket_state));
    if (state == NULL) {
        free(cert_text);
        free(key_text);
        jayess_throw(jayess_type_error_value("https.createServer failed to allocate TLS state"));
        return jayess_value_undefined();
    }
    state->handle = handle;
    state->ctx = SSL_CTX_new(TLS_server_method());
    if (state->ctx == NULL) {
        free(cert_text);
        free(key_text);
        jayess_std_tls_state_free(state, 0);
        jayess_throw(jayess_type_error_value("https.createServer failed to create TLS context"));
        return jayess_value_undefined();
    }
    if (SSL_CTX_use_certificate_file(state->ctx, cert_text, SSL_FILETYPE_PEM) != 1) {
        free(cert_text);
        free(key_text);
        jayess_std_tls_state_free(state, 0);
        jayess_throw(jayess_type_error_value("https.createServer failed to load server certificate"));
        return jayess_value_undefined();
    }
    if (SSL_CTX_use_PrivateKey_file(state->ctx, key_text, SSL_FILETYPE_PEM) != 1) {
        free(cert_text);
        free(key_text);
        jayess_std_tls_state_free(state, 0);
        jayess_throw(jayess_type_error_value("https.createServer failed to load private key"));
        return jayess_value_undefined();
    }
    if (SSL_CTX_check_private_key(state->ctx) != 1) {
        free(cert_text);
        free(key_text);
        jayess_std_tls_state_free(state, 0);
        jayess_throw(jayess_type_error_value("https.createServer certificate and private key do not match"));
        return jayess_value_undefined();
    }
    state->ssl = SSL_new(state->ctx);
    if (state->ssl == NULL) {
        free(cert_text);
        free(key_text);
        jayess_std_tls_state_free(state, 0);
        jayess_throw(jayess_type_error_value("https.createServer failed to create TLS session"));
        return jayess_value_undefined();
    }
    SSL_set_fd(state->ssl, handle);
    if (SSL_accept(state->ssl) != 1) {
        free(cert_text);
        free(key_text);
        jayess_std_tls_state_free(state, 0);
        jayess_std_socket_close_handle(handle);
        jayess_std_socket_set_handle(socket_value, JAYESS_INVALID_SOCKET);
        jayess_throw(jayess_type_error_value("https.createServer TLS handshake failed"));
        return jayess_value_undefined();
    }
    socket_value->as.object_value->native_handle = state;
    jayess_object_set_value(socket_value->as.object_value, "secure", jayess_value_from_bool(1));
    jayess_object_set_value(socket_value->as.object_value, "authorized", jayess_value_from_bool(0));
    jayess_object_set_value(socket_value->as.object_value, "backend", jayess_value_from_string("openssl"));
    jayess_object_set_value(socket_value->as.object_value, "protocol", jayess_value_from_string(SSL_get_version(state->ssl)));
    {
        const unsigned char *selected = NULL;
        unsigned int selected_length = 0;
        char negotiated_alpn[256];
        negotiated_alpn[0] = '\0';
        SSL_get0_alpn_selected(state->ssl, &selected, &selected_length);
        if (selected != NULL && selected_length > 0) {
            size_t copy_length = selected_length;
            if (copy_length >= sizeof(negotiated_alpn)) {
                copy_length = sizeof(negotiated_alpn) - 1;
            }
            memcpy(negotiated_alpn, selected, copy_length);
            negotiated_alpn[copy_length] = '\0';
        }
        jayess_object_set_value(socket_value->as.object_value, "alpnProtocol", jayess_value_from_string(negotiated_alpn));
    }
    free(cert_text);
    free(key_text);
    return socket_value;
#endif
}

static void jayess_std_socket_mark_closed(jayess_value *env) {
    if (env != NULL && env->kind == JAYESS_VALUE_OBJECT && env->as.object_value != NULL) {
        jayess_object_set_value(env->as.object_value, "connected", jayess_value_from_bool(0));
        jayess_object_set_value(env->as.object_value, "closed", jayess_value_from_bool(1));
        jayess_object_set_value(env->as.object_value, "readable", jayess_value_from_bool(0));
        jayess_object_set_value(env->as.object_value, "writable", jayess_value_from_bool(0));
    }
}

static void jayess_std_socket_emit_close(jayess_value *env) {
    jayess_value *already_emitted;
    if (env == NULL || env->kind != JAYESS_VALUE_OBJECT || env->as.object_value == NULL) {
        return;
    }
    already_emitted = jayess_object_get(env->as.object_value, "__jayess_socket_close_emitted");
    if (jayess_value_as_bool(already_emitted)) {
        return;
    }
    jayess_object_set_value(env->as.object_value, "__jayess_socket_close_emitted", jayess_value_from_bool(1));
    jayess_std_stream_emit(env, "close", jayess_value_undefined());
}

static void jayess_std_socket_close_native(jayess_value *env) {
    jayess_tls_socket_state *state = jayess_std_tls_state(env);
    if (state != NULL) {
        jayess_std_tls_state_free(state, 0);
        if (env != NULL && env->kind == JAYESS_VALUE_OBJECT && env->as.object_value != NULL) {
            env->as.object_value->native_handle = NULL;
        }
    }
}

static int jayess_std_socket_close_handle(jayess_socket_handle handle) {
    if (handle == JAYESS_INVALID_SOCKET) {
        return 1;
    }
#ifdef _WIN32
    return closesocket(handle) == 0;
#else
    return close(handle) == 0;
#endif
}

static jayess_value *jayess_std_socket_value_from_handle(jayess_socket_handle handle, const char *remote_address, int remote_port) {
    jayess_object *socket_object;
    if (handle == JAYESS_INVALID_SOCKET) {
        return jayess_value_undefined();
    }
    socket_object = jayess_object_new();
    if (socket_object == NULL) {
        jayess_std_socket_close_handle(handle);
        return jayess_value_from_object(NULL);
    }
    socket_object->socket_handle = handle;
    jayess_object_set_value(socket_object, "__jayess_std_kind", jayess_value_from_string("Socket"));
    jayess_object_set_value(socket_object, "connected", jayess_value_from_bool(1));
    jayess_object_set_value(socket_object, "closed", jayess_value_from_bool(0));
    jayess_object_set_value(socket_object, "readable", jayess_value_from_bool(1));
    jayess_object_set_value(socket_object, "writable", jayess_value_from_bool(1));
    jayess_object_set_value(socket_object, "remoteAddress", jayess_value_from_string(remote_address != NULL ? remote_address : ""));
    jayess_object_set_value(socket_object, "remotePort", jayess_value_from_number((double)remote_port));
    jayess_object_set_value(socket_object, "remoteFamily", jayess_value_from_number(0));
    jayess_object_set_value(socket_object, "localAddress", jayess_value_from_string(""));
    jayess_object_set_value(socket_object, "localPort", jayess_value_from_number(0));
    jayess_object_set_value(socket_object, "localFamily", jayess_value_from_number(0));
    jayess_object_set_value(socket_object, "bytesRead", jayess_value_from_number(0));
    jayess_object_set_value(socket_object, "bytesWritten", jayess_value_from_number(0));
    jayess_object_set_value(socket_object, "writableLength", jayess_value_from_number(0));
    jayess_object_set_value(socket_object, "writableHighWaterMark", jayess_value_from_number(JAYESS_STD_STREAM_DEFAULT_HIGH_WATER_MARK));
    jayess_object_set_value(socket_object, "writableNeedDrain", jayess_value_from_bool(0));
    jayess_object_set_value(socket_object, "timeout", jayess_value_from_number(0));
    jayess_object_set_value(socket_object, "errored", jayess_value_from_bool(0));
    jayess_object_set_value(socket_object, "secure", jayess_value_from_bool(0));
    jayess_object_set_value(socket_object, "authorized", jayess_value_from_bool(0));
    jayess_object_set_value(socket_object, "backend", jayess_value_from_string("tcp"));
    jayess_object_set_value(socket_object, "protocol", jayess_value_from_string(""));
    jayess_object_set_value(socket_object, "alpnProtocol", jayess_value_from_string(""));
    jayess_object_set_value(socket_object, "alpnProtocols", jayess_value_from_array(jayess_array_new()));
    return jayess_value_from_object(socket_object);
}

static jayess_value *jayess_std_datagram_socket_value_from_handle(jayess_socket_handle handle) {
    jayess_object *socket_object;
    if (handle == JAYESS_INVALID_SOCKET) {
        return jayess_value_undefined();
    }
    socket_object = jayess_object_new();
    if (socket_object == NULL) {
        jayess_std_socket_close_handle(handle);
        return jayess_value_from_object(NULL);
    }
    socket_object->socket_handle = handle;
    jayess_object_set_value(socket_object, "__jayess_std_kind", jayess_value_from_string("DatagramSocket"));
    jayess_object_set_value(socket_object, "closed", jayess_value_from_bool(0));
    jayess_object_set_value(socket_object, "localAddress", jayess_value_from_string(""));
    jayess_object_set_value(socket_object, "localPort", jayess_value_from_number(0));
    jayess_object_set_value(socket_object, "localFamily", jayess_value_from_number(0));
    jayess_object_set_value(socket_object, "bytesRead", jayess_value_from_number(0));
    jayess_object_set_value(socket_object, "bytesWritten", jayess_value_from_number(0));
    jayess_object_set_value(socket_object, "timeout", jayess_value_from_number(0));
    jayess_object_set_value(socket_object, "errored", jayess_value_from_bool(0));
    jayess_object_set_value(socket_object, "backend", jayess_value_from_string("udp"));
    jayess_object_set_value(socket_object, "protocol", jayess_value_from_string("udp"));
    jayess_object_set_value(socket_object, "broadcast", jayess_value_from_bool(0));
    jayess_object_set_value(socket_object, "multicastLoopback", jayess_value_from_bool(1));
    jayess_object_set_value(socket_object, "multicastInterface", jayess_value_from_string(""));
    return jayess_value_from_object(socket_object);
}

static void jayess_std_socket_set_local_endpoint(jayess_value *socket_value, jayess_socket_handle handle) {
    struct sockaddr_storage local_addr;
    char address[INET6_ADDRSTRLEN];
    int port = 0;
    int family = 0;
    void *addr_ptr = NULL;
#ifdef _WIN32
    int local_len = sizeof(local_addr);
#else
    socklen_t local_len = sizeof(local_addr);
#endif
    if (socket_value == NULL || socket_value->kind != JAYESS_VALUE_OBJECT || socket_value->as.object_value == NULL || handle == JAYESS_INVALID_SOCKET) {
        return;
    }
    memset(&local_addr, 0, sizeof(local_addr));
    address[0] = '\0';
    if (getsockname(handle, (struct sockaddr *)&local_addr, &local_len) != 0) {
        return;
    }
    if (local_addr.ss_family == AF_INET) {
        struct sockaddr_in *ipv4 = (struct sockaddr_in *)&local_addr;
        addr_ptr = &(ipv4->sin_addr);
        port = ntohs(ipv4->sin_port);
        family = 4;
    } else if (local_addr.ss_family == AF_INET6) {
        struct sockaddr_in6 *ipv6 = (struct sockaddr_in6 *)&local_addr;
        addr_ptr = &(ipv6->sin6_addr);
        port = ntohs(ipv6->sin6_port);
        family = 6;
    }
    if (addr_ptr == NULL || inet_ntop(local_addr.ss_family, addr_ptr, address, sizeof(address)) == NULL) {
        return;
    }
    jayess_object_set_value(socket_value->as.object_value, "localAddress", jayess_value_from_string(address));
    jayess_object_set_value(socket_value->as.object_value, "localPort", jayess_value_from_number((double)port));
    jayess_object_set_value(socket_value->as.object_value, "localFamily", jayess_value_from_number((double)family));
}

static void jayess_std_socket_set_remote_family(jayess_value *socket_value, int family) {
    if (socket_value == NULL || socket_value->kind != JAYESS_VALUE_OBJECT || socket_value->as.object_value == NULL) {
        return;
    }
    jayess_object_set_value(socket_value->as.object_value, "remoteFamily", jayess_value_from_number((double)family));
}

static jayess_value *jayess_std_tls_normalize_alpn_protocols(jayess_value *value) {
    jayess_array *result;
    int i;
    if (value == NULL || value->kind == JAYESS_VALUE_UNDEFINED || value->kind == JAYESS_VALUE_NULL) {
        return jayess_value_undefined();
    }
    result = jayess_array_new();
    if (result == NULL) {
        return jayess_value_undefined();
    }
    if (value->kind == JAYESS_VALUE_STRING) {
        const char *text = jayess_value_as_string(value);
        if (text == NULL || text[0] == '\0' || strlen(text) > 255) {
            jayess_throw(jayess_type_error_value("tls.connect alpnProtocols entries must be non-empty strings up to 255 bytes"));
            return NULL;
        }
        jayess_array_push_value(result, jayess_value_from_string(text));
        return jayess_value_from_array(result);
    }
    if (value->kind != JAYESS_VALUE_ARRAY || value->as.array_value == NULL) {
        jayess_throw(jayess_type_error_value("tls.connect alpnProtocols must be a string or array of strings"));
        return NULL;
    }
    for (i = 0; i < value->as.array_value->count; i++) {
        char *text = jayess_value_stringify(value->as.array_value->values[i]);
        size_t length = text != NULL ? strlen(text) : 0;
        if (text == NULL || text[0] == '\0' || length > 255) {
            free(text);
            jayess_throw(jayess_type_error_value("tls.connect alpnProtocols entries must be non-empty strings up to 255 bytes"));
            return NULL;
        }
        jayess_array_push_value(result, jayess_value_from_string(text));
        free(text);
    }
    return jayess_value_from_array(result);
}

static int jayess_std_tls_build_alpn_wire(jayess_value *protocols_value, unsigned char **out_buffer, size_t *out_length) {
    size_t total_length = 0;
    int i;
    jayess_array *protocols;
    unsigned char *buffer;
    size_t offset = 0;
    if (out_buffer == NULL || out_length == NULL) {
        return 0;
    }
    *out_buffer = NULL;
    *out_length = 0;
    if (protocols_value == NULL || protocols_value->kind != JAYESS_VALUE_ARRAY || protocols_value->as.array_value == NULL) {
        return 1;
    }
    protocols = protocols_value->as.array_value;
    if (protocols->count == 0) {
        return 1;
    }
    for (i = 0; i < protocols->count; i++) {
        const char *text = jayess_value_as_string(protocols->values[i]);
        size_t length = text != NULL ? strlen(text) : 0;
        if (text == NULL || text[0] == '\0' || length > 255) {
            return 0;
        }
        total_length += 1 + length;
    }
    buffer = (unsigned char *)malloc(total_length);
    if (buffer == NULL) {
        return 0;
    }
    for (i = 0; i < protocols->count; i++) {
        const char *text = jayess_value_as_string(protocols->values[i]);
        size_t length = strlen(text);
        buffer[offset++] = (unsigned char)length;
        memcpy(buffer + offset, text, length);
        offset += length;
    }
    *out_buffer = buffer;
    *out_length = total_length;
    return 1;
}

static void jayess_std_https_copy_tls_request_settings(jayess_object *target, jayess_object *source) {
    static const char *keys[] = {
        "rejectUnauthorized",
        "serverName",
        "caFile",
        "caPath",
        "trustSystem"
    };
    int i;
    if (target == NULL || source == NULL) {
        return;
    }
    for (i = 0; i < (int)(sizeof(keys) / sizeof(keys[0])); i++) {
        jayess_value *value = jayess_object_get(source, keys[i]);
        if (value != NULL) {
            jayess_object_set_value(target, keys[i], value);
        }
    }
}

static void jayess_std_tls_array_push_prefixed(jayess_array *array, const char *prefix, const char *value) {
    size_t prefix_len;
    size_t value_len;
    char *text;
    if (array == NULL || prefix == NULL || value == NULL || value[0] == '\0') {
        return;
    }
    prefix_len = strlen(prefix);
    value_len = strlen(value);
    text = (char *)malloc(prefix_len + value_len + 1);
    if (text == NULL) {
        return;
    }
    memcpy(text, prefix, prefix_len);
    memcpy(text + prefix_len, value, value_len + 1);
    jayess_array_push_value(array, jayess_value_from_string(text));
    free(text);
}

static jayess_value *jayess_std_tls_subject_alt_names(jayess_value *env) {
    jayess_tls_socket_state *state = jayess_std_tls_state(env);
    jayess_array *names = jayess_array_new();
    if (names == NULL) {
        return jayess_value_from_array(NULL);
    }
    if (state == NULL) {
        return jayess_value_from_array(names);
    }
#ifdef _WIN32
    {
        PCCERT_CONTEXT cert = NULL;
        SECURITY_STATUS status = QueryContextAttributes(&state->context, SECPKG_ATTR_REMOTE_CERT_CONTEXT, &cert);
        if (status == SEC_E_OK && cert != NULL) {
            PCERT_EXTENSION extension = CertFindExtension(szOID_SUBJECT_ALT_NAME2, cert->pCertInfo->cExtension, cert->pCertInfo->rgExtension);
            if (extension != NULL && extension->Value.pbData != NULL && extension->Value.cbData > 0) {
                DWORD decoded_size = 0;
                if (CryptDecodeObject(X509_ASN_ENCODING | PKCS_7_ASN_ENCODING, X509_ALTERNATE_NAME, extension->Value.pbData, extension->Value.cbData, 0, NULL, &decoded_size) && decoded_size > 0) {
                    CERT_ALT_NAME_INFO *info = (CERT_ALT_NAME_INFO *)malloc(decoded_size);
                    if (info != NULL) {
                        if (CryptDecodeObject(X509_ASN_ENCODING | PKCS_7_ASN_ENCODING, X509_ALTERNATE_NAME, extension->Value.pbData, extension->Value.cbData, 0, info, &decoded_size)) {
                            DWORD i;
                            for (i = 0; i < info->cAltEntry; i++) {
                                CERT_ALT_NAME_ENTRY *entry = &info->rgAltEntry[i];
                                if (entry->dwAltNameChoice == CERT_ALT_NAME_DNS_NAME && entry->pwszDNSName != NULL) {
                                    char *dns = jayess_wide_to_utf8(entry->pwszDNSName);
                                    if (dns != NULL) {
                                        jayess_std_tls_array_push_prefixed(names, "DNS:", dns);
                                        free(dns);
                                    }
                                } else if (entry->dwAltNameChoice == CERT_ALT_NAME_URL && entry->pwszURL != NULL) {
                                    char *uri = jayess_wide_to_utf8(entry->pwszURL);
                                    if (uri != NULL) {
                                        jayess_std_tls_array_push_prefixed(names, "URI:", uri);
                                        free(uri);
                                    }
                                } else if (entry->dwAltNameChoice == CERT_ALT_NAME_RFC822_NAME && entry->pwszRfc822Name != NULL) {
                                    char *email = jayess_wide_to_utf8(entry->pwszRfc822Name);
                                    if (email != NULL) {
                                        jayess_std_tls_array_push_prefixed(names, "EMAIL:", email);
                                        free(email);
                                    }
                                } else if (entry->dwAltNameChoice == CERT_ALT_NAME_IP_ADDRESS && entry->IPAddress.pbData != NULL) {
                                    char address[INET6_ADDRSTRLEN];
                                    const void *addr_ptr = NULL;
                                    int family = 0;
                                    address[0] = '\0';
                                    if (entry->IPAddress.cbData == 4) {
                                        family = AF_INET;
                                        addr_ptr = entry->IPAddress.pbData;
                                    } else if (entry->IPAddress.cbData == 16) {
                                        family = AF_INET6;
                                        addr_ptr = entry->IPAddress.pbData;
                                    }
                                    if (addr_ptr != NULL && inet_ntop(family, addr_ptr, address, sizeof(address)) != NULL) {
                                        jayess_std_tls_array_push_prefixed(names, "IP:", address);
                                    }
                                }
                            }
                        }
                        free(info);
                    }
                }
            }
            CertFreeCertificateContext(cert);
        }
    }
#else
    {
        X509 *cert = SSL_get_peer_certificate(state->ssl);
        if (cert != NULL) {
            GENERAL_NAMES *general_names = X509_get_ext_d2i(cert, NID_subject_alt_name, NULL, NULL);
            if (general_names != NULL) {
                int count = sk_GENERAL_NAME_num(general_names);
                int i;
                for (i = 0; i < count; i++) {
                    GENERAL_NAME *name = sk_GENERAL_NAME_value(general_names, i);
                    if (name == NULL) {
                        continue;
                    }
                    if (name->type == GEN_DNS || name->type == GEN_URI || name->type == GEN_EMAIL) {
                        const unsigned char *data = ASN1_STRING_get0_data(name->d.ia5);
                        int length = ASN1_STRING_length(name->d.ia5);
                        char *text;
                        const char *prefix = name->type == GEN_DNS ? "DNS:" : (name->type == GEN_URI ? "URI:" : "EMAIL:");
                        if (data == NULL || length <= 0) {
                            continue;
                        }
                        text = (char *)malloc((size_t)length + 1);
                        if (text == NULL) {
                            continue;
                        }
                        memcpy(text, data, (size_t)length);
                        text[length] = '\0';
                        jayess_std_tls_array_push_prefixed(names, prefix, text);
                        free(text);
                    } else if (name->type == GEN_IPADD) {
                        char address[INET6_ADDRSTRLEN];
                        const unsigned char *data = ASN1_STRING_get0_data(name->d.ip);
                        int length = ASN1_STRING_length(name->d.ip);
                        int family = 0;
                        address[0] = '\0';
                        if (data == NULL) {
                            continue;
                        }
                        if (length == 4) {
                            family = AF_INET;
                        } else if (length == 16) {
                            family = AF_INET6;
                        }
                        if (family != 0 && inet_ntop(family, data, address, sizeof(address)) != NULL) {
                            jayess_std_tls_array_push_prefixed(names, "IP:", address);
                        }
                    }
                }
                GENERAL_NAMES_free(general_names);
            }
            X509_free(cert);
        }
    }
#endif
    return jayess_value_from_array(names);
}

#ifdef _WIN32
static int jayess_std_windows_add_encoded_certificate(HCERTSTORE store, const unsigned char *data, DWORD length) {
    if (store == NULL || data == NULL || length == 0) {
        return 0;
    }
    return CertAddEncodedCertificateToStore(
        store,
        X509_ASN_ENCODING | PKCS_7_ASN_ENCODING,
        data,
        length,
        CERT_STORE_ADD_REPLACE_EXISTING,
        NULL);
}

static int jayess_std_windows_load_certificates_from_file(HCERTSTORE store, const char *path) {
    FILE *file;
    long length;
    char *buffer;
    char *cursor;
    int loaded = 0;
    if (store == NULL || path == NULL || path[0] == '\0') {
        return 0;
    }
    file = fopen(path, "rb");
    if (file == NULL) {
        return 0;
    }
    if (fseek(file, 0, SEEK_END) != 0) {
        fclose(file);
        return 0;
    }
    length = ftell(file);
    if (length <= 0) {
        fclose(file);
        return 0;
    }
    if (fseek(file, 0, SEEK_SET) != 0) {
        fclose(file);
        return 0;
    }
    buffer = (char *)malloc((size_t)length + 1);
    if (buffer == NULL) {
        fclose(file);
        return 0;
    }
    if (fread(buffer, 1, (size_t)length, file) != (size_t)length) {
        free(buffer);
        fclose(file);
        return 0;
    }
    buffer[length] = '\0';
    fclose(file);

    cursor = buffer;
    while (1) {
        char *begin = strstr(cursor, "-----BEGIN CERTIFICATE-----");
        if (begin == NULL) {
            break;
        }
        char *end = strstr(begin, "-----END CERTIFICATE-----");
        DWORD decoded_length = 0;
        BYTE *decoded = NULL;
        if (end == NULL) {
            break;
        }
        end += (int)strlen("-----END CERTIFICATE-----");
        while (*end == '\r' || *end == '\n') {
            end++;
        }
        if (CryptStringToBinaryA(begin, (DWORD)(end - begin), CRYPT_STRING_BASE64HEADER, NULL, &decoded_length, NULL, NULL) && decoded_length > 0) {
            decoded = (BYTE *)malloc(decoded_length);
            if (decoded != NULL && CryptStringToBinaryA(begin, (DWORD)(end - begin), CRYPT_STRING_BASE64HEADER, decoded, &decoded_length, NULL, NULL)) {
                if (jayess_std_windows_add_encoded_certificate(store, decoded, decoded_length)) {
                    loaded++;
                }
            }
            free(decoded);
        }
        cursor = end;
    }
    if (loaded == 0) {
        if (jayess_std_windows_add_encoded_certificate(store, (const unsigned char *)buffer, (DWORD)length)) {
            loaded = 1;
        }
    }
    free(buffer);
    return loaded > 0;
}

static int jayess_std_windows_load_certificates_from_path(HCERTSTORE store, const char *path) {
    char pattern[MAX_PATH];
    WIN32_FIND_DATAA find_data;
    HANDLE find_handle;
    int loaded = 0;
    if (store == NULL || path == NULL || path[0] == '\0') {
        return 0;
    }
    if (jayess_path_is_separator(path[strlen(path) - 1])) {
        snprintf(pattern, sizeof(pattern), "%s*", path);
    } else {
        snprintf(pattern, sizeof(pattern), "%s\\*", path);
    }
    find_handle = FindFirstFileA(pattern, &find_data);
    if (find_handle == INVALID_HANDLE_VALUE) {
        return 0;
    }
    do {
        char full_path[MAX_PATH];
        if (strcmp(find_data.cFileName, ".") == 0 || strcmp(find_data.cFileName, "..") == 0) {
            continue;
        }
        if ((find_data.dwFileAttributes & FILE_ATTRIBUTE_DIRECTORY) != 0) {
            continue;
        }
        if (jayess_path_is_separator(path[strlen(path) - 1])) {
            snprintf(full_path, sizeof(full_path), "%s%s", path, find_data.cFileName);
        } else {
            snprintf(full_path, sizeof(full_path), "%s\\%s", path, find_data.cFileName);
        }
        if (jayess_std_windows_load_certificates_from_file(store, full_path)) {
            loaded = 1;
        }
    } while (FindNextFileA(find_handle, &find_data));
    FindClose(find_handle);
    return loaded;
}

static int jayess_std_windows_validate_tls_certificate(jayess_tls_socket_state *state, const char *server_name, const char *ca_file, const char *ca_path, int trust_system) {
    PCCERT_CONTEXT cert = NULL;
    HCERTSTORE custom_store = NULL;
    HCERTSTORE collection_store = NULL;
    HCERTSTORE system_root = NULL;
    HCERTSTORE system_trusted_people = NULL;
    HCERTCHAINENGINE engine = NULL;
    CERT_CHAIN_ENGINE_CONFIG engine_config;
    CERT_CHAIN_PARA chain_para;
    PCCERT_CHAIN_CONTEXT chain = NULL;
    HTTPSPolicyCallbackData policy_data;
    CERT_CHAIN_POLICY_PARA policy_para;
    CERT_CHAIN_POLICY_STATUS policy_status;
    wchar_t *server_name_wide = NULL;
    int ok = 0;
    int has_custom_trust = 0;

    if (state == NULL || server_name == NULL || server_name[0] == '\0') {
        return 0;
    }
    if (QueryContextAttributes(&state->context, SECPKG_ATTR_REMOTE_CERT_CONTEXT, &cert) != SEC_E_OK || cert == NULL) {
        return 0;
    }
    if ((ca_file != NULL && ca_file[0] != '\0') || (ca_path != NULL && ca_path[0] != '\0') || !trust_system) {
        custom_store = CertOpenStore(CERT_STORE_PROV_MEMORY, 0, 0, CERT_STORE_CREATE_NEW_FLAG, NULL);
        if (custom_store == NULL) {
            goto cleanup;
        }
        if (ca_file != NULL && ca_file[0] != '\0') {
            has_custom_trust = jayess_std_windows_load_certificates_from_file(custom_store, ca_file) || has_custom_trust;
        }
        if (ca_path != NULL && ca_path[0] != '\0') {
            has_custom_trust = jayess_std_windows_load_certificates_from_path(custom_store, ca_path) || has_custom_trust;
        }
        if (!trust_system && !has_custom_trust) {
            goto cleanup;
        }
        collection_store = CertOpenStore(CERT_STORE_PROV_COLLECTION, 0, 0, CERT_STORE_CREATE_NEW_FLAG, NULL);
        if (collection_store == NULL) {
            goto cleanup;
        }
        if (trust_system) {
            system_root = CertOpenSystemStoreA(0, "ROOT");
            system_trusted_people = CertOpenSystemStoreA(0, "TrustedPeople");
            if (system_root != NULL) {
                CertAddStoreToCollection(collection_store, system_root, 0, 0);
            }
            if (system_trusted_people != NULL) {
                CertAddStoreToCollection(collection_store, system_trusted_people, 0, 0);
            }
        }
        if (custom_store != NULL) {
            CertAddStoreToCollection(collection_store, custom_store, 0, 0);
        }
        memset(&engine_config, 0, sizeof(engine_config));
        engine_config.cbSize = sizeof(engine_config);
        engine_config.hExclusiveRoot = collection_store;
#if (NTDDI_VERSION >= NTDDI_WIN8)
        engine_config.dwExclusiveFlags = CERT_CHAIN_EXCLUSIVE_ENABLE_CA_FLAG;
#endif
        if (!CertCreateCertificateChainEngine(&engine_config, &engine)) {
            goto cleanup;
        }
    }

    memset(&chain_para, 0, sizeof(chain_para));
    chain_para.cbSize = sizeof(chain_para);
    if (!CertGetCertificateChain(engine, cert, NULL, cert->hCertStore, &chain_para, 0, NULL, &chain)) {
        goto cleanup;
    }

    server_name_wide = jayess_utf8_to_wide(server_name);
    if (server_name_wide == NULL) {
        goto cleanup;
    }
    memset(&policy_data, 0, sizeof(policy_data));
    policy_data.cbStruct = sizeof(policy_data);
    policy_data.dwAuthType = AUTHTYPE_SERVER;
    policy_data.pwszServerName = server_name_wide;
    memset(&policy_para, 0, sizeof(policy_para));
    policy_para.cbSize = sizeof(policy_para);
    policy_para.pvExtraPolicyPara = &policy_data;
    memset(&policy_status, 0, sizeof(policy_status));
    policy_status.cbSize = sizeof(policy_status);
    if (!CertVerifyCertificateChainPolicy(CERT_CHAIN_POLICY_SSL, chain, &policy_para, &policy_status)) {
        goto cleanup;
    }
    ok = policy_status.dwError == 0;

cleanup:
    free(server_name_wide);
    if (chain != NULL) {
        CertFreeCertificateChain(chain);
    }
    if (engine != NULL) {
        CertFreeCertificateChainEngine(engine);
    }
    if (system_trusted_people != NULL) {
        CertCloseStore(system_trusted_people, 0);
    }
    if (system_root != NULL) {
        CertCloseStore(system_root, 0);
    }
    if (collection_store != NULL) {
        CertCloseStore(collection_store, 0);
    }
    if (custom_store != NULL) {
        CertCloseStore(custom_store, 0);
    }
    if (cert != NULL) {
        CertFreeCertificateContext(cert);
    }
    return ok;
}
#endif

#ifdef _WIN32
static void *jayess_std_tls_build_schannel_alpn_buffer(const unsigned char *wire, size_t wire_length, unsigned long *buffer_length) {
    size_t total_size;
    SEC_APPLICATION_PROTOCOLS *protocols;
    if (buffer_length == NULL) {
        return NULL;
    }
    *buffer_length = 0;
    if (wire == NULL || wire_length == 0) {
        return NULL;
    }
    total_size = FIELD_OFFSET(SEC_APPLICATION_PROTOCOLS, ProtocolLists) +
        FIELD_OFFSET(SEC_APPLICATION_PROTOCOL_LIST, ProtocolList) + wire_length;
    protocols = (SEC_APPLICATION_PROTOCOLS *)calloc(1, total_size);
    if (protocols == NULL) {
        return NULL;
    }
    protocols->ProtocolListsSize = (unsigned long)(FIELD_OFFSET(SEC_APPLICATION_PROTOCOL_LIST, ProtocolList) + wire_length);
    protocols->ProtocolLists[0].ProtoNegoExt = SecApplicationProtocolNegotiationExt_ALPN;
    protocols->ProtocolLists[0].ProtocolListSize = (unsigned short)wire_length;
    memcpy(protocols->ProtocolLists[0].ProtocolList, wire, wire_length);
    *buffer_length = (unsigned long)total_size;
    return protocols;
}

static const char *jayess_std_tls_windows_protocol_name(DWORD protocol) {
#ifdef SP_PROT_TLS1_3_CLIENT
    if (protocol & (SP_PROT_TLS1_3_CLIENT | SP_PROT_TLS1_3_SERVER)) {
        return "TLSv1.3";
    }
#endif
    if (protocol & (SP_PROT_TLS1_2_CLIENT | SP_PROT_TLS1_2_SERVER)) {
        return "TLSv1.2";
    }
    if (protocol & (SP_PROT_TLS1_1_CLIENT | SP_PROT_TLS1_1_SERVER)) {
        return "TLSv1.1";
    }
    if (protocol & (SP_PROT_TLS1_CLIENT | SP_PROT_TLS1_SERVER)) {
        return "TLSv1.0";
    }
#ifdef SP_PROT_SSL3_CLIENT
    if (protocol & (SP_PROT_SSL3_CLIENT | SP_PROT_SSL3_SERVER)) {
        return "SSLv3";
    }
#endif
    return "";
}
#endif

static void jayess_std_fs_watch_emit_close(jayess_value *env) {
    jayess_value *already_emitted;
    if (env == NULL || env->kind != JAYESS_VALUE_OBJECT || env->as.object_value == NULL) {
        return;
    }
    already_emitted = jayess_object_get(env->as.object_value, "__jayess_watcher_close_emitted");
    if (jayess_value_as_bool(already_emitted)) {
        return;
    }
    jayess_object_set_value(env->as.object_value, "__jayess_watcher_close_emitted", jayess_value_from_bool(1));
    jayess_std_stream_emit(env, "close", jayess_value_undefined());
}

static jayess_value *jayess_std_fs_watch_poll_method(jayess_value *env) {
    jayess_fs_watch_state *state = jayess_std_fs_watch_state(env);
    int exists;
    int is_dir;
    double size;
    double mtime_ms;
    int changed;
    jayess_value *event;
    if (state == NULL || state->closed) {
        return jayess_value_null();
    }
    jayess_fs_watch_snapshot_text(state->path, &exists, &is_dir, &size, &mtime_ms);
    changed = exists != state->exists || is_dir != state->is_dir || size != state->size || mtime_ms != state->mtime_ms;
    if (!changed) {
        return jayess_value_null();
    }
    state->exists = exists;
    state->is_dir = is_dir;
    state->size = size;
    state->mtime_ms = mtime_ms;
    jayess_std_fs_watch_apply_snapshot(env, exists, is_dir, size, mtime_ms);
    event = jayess_std_fs_watch_event_value(state);
    jayess_std_stream_emit(env, "change", event);
    return event;
}

static jayess_value *jayess_std_fs_watch_poll_async_tick(jayess_value *env) {
    jayess_value *watcher;
    jayess_value *promise;
    jayess_value *callback;
    jayess_value *result;
    double deadline;
    if (env == NULL || env->kind != JAYESS_VALUE_OBJECT || env->as.object_value == NULL) {
        return jayess_value_undefined();
    }
    watcher = jayess_object_get(env->as.object_value, "watcher");
    promise = jayess_object_get(env->as.object_value, "promise");
    callback = jayess_object_get(env->as.object_value, "callback");
    deadline = jayess_value_to_number(jayess_object_get(env->as.object_value, "deadlineMs"));
    if (promise == NULL || promise->kind != JAYESS_VALUE_OBJECT || !jayess_std_kind_is(promise, "Promise") || !jayess_promise_is_state(promise, "pending")) {
        return jayess_value_undefined();
    }
    result = jayess_std_fs_watch_poll_method(watcher);
    if (result != NULL && result->kind != JAYESS_VALUE_NULL) {
        jayess_promise_settle(promise, "fulfilled", result);
        return jayess_value_undefined();
    }
    if (watcher == NULL || watcher->kind != JAYESS_VALUE_OBJECT || jayess_value_as_bool(jayess_value_get_member(watcher, "closed"))) {
        jayess_promise_settle(promise, "fulfilled", jayess_value_null());
        return jayess_value_undefined();
    }
    if (deadline >= 0 && jayess_now_ms() >= deadline) {
        jayess_promise_settle(promise, "fulfilled", jayess_value_null());
        return jayess_value_undefined();
    }
    jayess_set_timeout(callback, jayess_value_from_number(10));
    return jayess_value_undefined();
}

static jayess_value *jayess_std_fs_watch_poll_async_method(jayess_value *env, jayess_value *timeout_ms) {
    jayess_value *immediate;
    jayess_value *promise;
    jayess_object *state;
    jayess_value *state_value;
    jayess_value *callback;
    double timeout = -1.0;
    if (timeout_ms != NULL && !jayess_value_is_nullish(timeout_ms)) {
        timeout = jayess_value_to_number(timeout_ms);
        if (timeout < 0) {
            timeout = 0;
        }
    }
    immediate = jayess_std_fs_watch_poll_method(env);
    if (immediate != NULL && immediate->kind != JAYESS_VALUE_NULL) {
        return jayess_std_promise_resolve(immediate);
    }
    if (env == NULL || env->kind != JAYESS_VALUE_OBJECT || env->as.object_value == NULL || jayess_value_as_bool(jayess_object_get(env->as.object_value, "closed"))) {
        return jayess_std_promise_resolve(jayess_value_null());
    }
    promise = jayess_std_promise_pending();
    state = jayess_object_new();
    if (state == NULL) {
        jayess_promise_settle(promise, "rejected", jayess_type_error_value("failed to allocate watcher async state"));
        return promise;
    }
    state_value = jayess_value_from_object(state);
    jayess_object_set_value(state, "watcher", env);
    jayess_object_set_value(state, "promise", promise);
    jayess_object_set_value(state, "deadlineMs", jayess_value_from_number(timeout >= 0 ? jayess_now_ms() + timeout : -1.0));
    callback = jayess_value_from_function((void *)jayess_std_fs_watch_poll_async_tick, state_value, "__jayess_fs_watch_poll_async_tick", NULL, 0, 0);
    jayess_object_set_value(state, "callback", callback);
    jayess_set_timeout(callback, jayess_value_from_number(10));
    return promise;
}

static jayess_value *jayess_std_fs_watch_close_method(jayess_value *env) {
    jayess_fs_watch_state *state = jayess_std_fs_watch_state(env);
    if (state != NULL && !state->closed) {
        state->closed = 1;
        free(state->path);
        free(state);
        if (env != NULL && env->kind == JAYESS_VALUE_OBJECT && env->as.object_value != NULL) {
            env->as.object_value->native_handle = NULL;
        }
    }
    if (env != NULL && env->kind == JAYESS_VALUE_OBJECT && env->as.object_value != NULL) {
        jayess_object_set_value(env->as.object_value, "closed", jayess_value_from_bool(1));
    }
    jayess_std_fs_watch_emit_close(env);
    return env != NULL ? env : jayess_value_undefined();
}

static jayess_value *jayess_std_fs_watch_on_method(jayess_value *env, jayess_value *event, jayess_value *callback) {
    char *event_text = jayess_value_stringify(event);
    if (event_text == NULL || callback == NULL || callback->kind != JAYESS_VALUE_FUNCTION) {
        free(event_text);
        return env != NULL ? env : jayess_value_undefined();
    }
    if (env == NULL || env->kind != JAYESS_VALUE_OBJECT || env->as.object_value == NULL) {
        free(event_text);
        return jayess_value_undefined();
    }
    if (strcmp(event_text, "error") == 0) {
        jayess_std_stream_register_error_handler(env, callback);
    } else if (strcmp(event_text, "close") == 0) {
        jayess_std_stream_on(env, "close", callback);
        if (jayess_value_as_bool(jayess_object_get(env->as.object_value, "closed"))) {
            jayess_value_call_one(callback, jayess_value_undefined());
        }
    } else if (strcmp(event_text, "change") == 0) {
        jayess_std_stream_on(env, "change", callback);
    }
    free(event_text);
    return env != NULL ? env : jayess_value_undefined();
}

static jayess_value *jayess_std_fs_watch_once_method(jayess_value *env, jayess_value *event, jayess_value *callback) {
    char *event_text = jayess_value_stringify(event);
    if (event_text == NULL || callback == NULL || callback->kind != JAYESS_VALUE_FUNCTION) {
        free(event_text);
        return env != NULL ? env : jayess_value_undefined();
    }
    if (env == NULL || env->kind != JAYESS_VALUE_OBJECT || env->as.object_value == NULL) {
        free(event_text);
        return jayess_value_undefined();
    }
    if (strcmp(event_text, "error") == 0) {
        jayess_std_stream_register_error_once(env, callback);
    } else if (strcmp(event_text, "close") == 0) {
        if (jayess_value_as_bool(jayess_object_get(env->as.object_value, "closed"))) {
            jayess_value_call_one(callback, jayess_value_undefined());
        } else {
            jayess_std_stream_once(env, "close", callback);
        }
    } else if (strcmp(event_text, "change") == 0) {
        jayess_std_stream_once(env, "change", callback);
    }
    free(event_text);
    return env != NULL ? env : jayess_value_undefined();
}

static void jayess_std_read_stream_mark_ended(jayess_value *env) {
    if (env != NULL && env->kind == JAYESS_VALUE_OBJECT && env->as.object_value != NULL) {
        jayess_object_set_value(env->as.object_value, "readableEnded", jayess_value_from_bool(1));
    }
}

static void jayess_std_read_stream_emit_end(jayess_value *env) {
    jayess_std_read_stream_mark_ended(env);
    if (env == NULL || env->kind != JAYESS_VALUE_OBJECT || env->as.object_value == NULL) {
        return;
    }
    jayess_std_stream_emit(env, "end", jayess_value_undefined());
}

static jayess_value *jayess_std_read_stream_read_chunk(jayess_value *env, jayess_value *size_value) {
    FILE *file = jayess_std_stream_file(env);
    int requested = jayess_std_stream_requested_size(size_value, 4095);
    char *buffer;
    size_t read_count;
    jayess_value *result;
    if (file == NULL) {
        if (env != NULL && env->kind == JAYESS_VALUE_OBJECT) {
            jayess_object_set_value(env->as.object_value, "closed", jayess_value_from_bool(1));
            jayess_std_read_stream_mark_ended(env);
        }
        return jayess_value_undefined();
    }
    buffer = (char *)malloc((size_t)requested + 1);
    if (buffer == NULL) {
        jayess_std_stream_emit_error(env, "failed to allocate stream read buffer");
        return jayess_value_undefined();
    }
    read_count = fread(buffer, 1, (size_t)requested, file);
    if (read_count == 0) {
        free(buffer);
        if (feof(file)) {
            jayess_std_read_stream_emit_end(env);
            return jayess_value_null();
        }
        jayess_std_stream_emit_error(env, "failed to read from stream");
        return jayess_value_undefined();
    }
    buffer[read_count] = '\0';
    result = jayess_value_from_string(buffer);
    free(buffer);
    return result;
}

static jayess_value *jayess_std_read_stream_read_method(jayess_value *env, jayess_value *size_value) {
    return jayess_std_read_stream_read_chunk(env, size_value);
}

static jayess_value *jayess_std_read_stream_read_bytes_method(jayess_value *env, jayess_value *size_value) {
    FILE *file = jayess_std_stream_file(env);
    int requested = jayess_std_stream_requested_size(size_value, 4095);
    unsigned char *buffer;
    size_t read_count;
    jayess_value *array_buffer;
    jayess_value *view;
    jayess_array *bytes;
    int i;
    if (file == NULL) {
        if (env != NULL && env->kind == JAYESS_VALUE_OBJECT) {
            jayess_object_set_value(env->as.object_value, "closed", jayess_value_from_bool(1));
            jayess_std_read_stream_mark_ended(env);
        }
        return jayess_value_undefined();
    }
    buffer = (unsigned char *)malloc((size_t)requested);
    if (buffer == NULL) {
        jayess_std_stream_emit_error(env, "failed to allocate stream read buffer");
        return jayess_value_undefined();
    }
    read_count = fread(buffer, 1, (size_t)requested, file);
    if (read_count == 0) {
        free(buffer);
        if (feof(file)) {
            jayess_std_read_stream_emit_end(env);
            return jayess_value_null();
        }
        jayess_std_stream_emit_error(env, "failed to read from stream");
        return jayess_value_undefined();
    }
    array_buffer = jayess_std_array_buffer_new(jayess_value_from_number((double)read_count));
    view = jayess_std_uint8_array_new(array_buffer);
    bytes = jayess_std_bytes_slot(view);
    if (bytes == NULL) {
        free(buffer);
        return jayess_value_undefined();
    }
    for (i = 0; i < (int)read_count; i++) {
        jayess_array_set_value(bytes, i, jayess_value_from_number((double)buffer[i]));
    }
    free(buffer);
    return view;
}

static jayess_value *jayess_std_read_stream_close_method(jayess_value *env) {
    FILE *file = jayess_std_stream_file(env);
    if (file != NULL) {
        fclose(file);
        jayess_std_stream_set_file(env, NULL);
        if (env != NULL && env->kind == JAYESS_VALUE_OBJECT) {
            jayess_object_set_value(env->as.object_value, "closed", jayess_value_from_bool(1));
            jayess_std_read_stream_mark_ended(env);
        }
    }
    return jayess_value_undefined();
}

static jayess_value *jayess_std_read_stream_on_method(jayess_value *env, jayess_value *event, jayess_value *callback) {
    char *event_text = jayess_value_stringify(event);
    if (event_text == NULL || callback == NULL || callback->kind != JAYESS_VALUE_FUNCTION) {
        free(event_text);
        return env != NULL ? env : jayess_value_undefined();
    }
    if (env == NULL || env->kind != JAYESS_VALUE_OBJECT || env->as.object_value == NULL) {
        free(event_text);
        return jayess_value_undefined();
    }
    if (strcmp(event_text, "end") == 0) {
        jayess_std_stream_on(env, "end", callback);
        if (jayess_value_as_bool(jayess_object_get(env->as.object_value, "readableEnded"))) {
            jayess_value_call_one(callback, jayess_value_undefined());
        }
    } else if (strcmp(event_text, "error") == 0) {
        jayess_std_stream_register_error_handler(env, callback);
    } else if (strcmp(event_text, "data") == 0) {
        while (1) {
            jayess_value *chunk = jayess_std_read_stream_read_chunk(env, jayess_value_undefined());
            if (chunk == NULL || chunk->kind == JAYESS_VALUE_NULL || chunk->kind == JAYESS_VALUE_UNDEFINED) {
                break;
            }
            jayess_value_call_one(callback, chunk);
            if (jayess_has_exception()) {
                break;
            }
        }
    }
    free(event_text);
    return env;
}

static jayess_value *jayess_std_read_stream_once_method(jayess_value *env, jayess_value *event, jayess_value *callback) {
    char *event_text = jayess_value_stringify(event);
    if (event_text == NULL || callback == NULL || callback->kind != JAYESS_VALUE_FUNCTION) {
        free(event_text);
        return env != NULL ? env : jayess_value_undefined();
    }
    if (env == NULL || env->kind != JAYESS_VALUE_OBJECT || env->as.object_value == NULL) {
        free(event_text);
        return jayess_value_undefined();
    }
    if (strcmp(event_text, "end") == 0) {
        if (jayess_value_as_bool(jayess_object_get(env->as.object_value, "readableEnded"))) {
            jayess_value_call_one(callback, jayess_value_undefined());
        } else {
            jayess_std_stream_once(env, "end", callback);
        }
    } else if (strcmp(event_text, "error") == 0) {
        jayess_std_stream_register_error_once(env, callback);
    } else if (strcmp(event_text, "data") == 0) {
        jayess_value *chunk = jayess_std_read_stream_read_chunk(env, jayess_value_undefined());
        if (chunk != NULL && chunk->kind != JAYESS_VALUE_NULL && chunk->kind != JAYESS_VALUE_UNDEFINED) {
            jayess_value_call_one(callback, chunk);
        }
    }
    free(event_text);
    return env;
}

static jayess_value *jayess_std_read_stream_pipe_method(jayess_value *env, jayess_value *destination) {
    if (destination == NULL || destination->kind != JAYESS_VALUE_OBJECT) {
        return destination != NULL ? destination : jayess_value_undefined();
    }
    while (1) {
        jayess_value *chunk;
        if (jayess_std_kind_is(destination, "CompressionStream")) {
            chunk = jayess_std_read_stream_read_bytes_method(env, jayess_value_undefined());
        } else {
            chunk = jayess_std_read_stream_read_chunk(env, jayess_value_undefined());
        }
        if (chunk == NULL || chunk->kind == JAYESS_VALUE_NULL || chunk->kind == JAYESS_VALUE_UNDEFINED) {
            break;
        }
        jayess_std_writable_write(destination, chunk);
        if (jayess_has_exception()) {
            break;
        }
    }
    jayess_std_writable_end(destination);
    return destination;
}

static jayess_value *jayess_std_http_body_stream_read_method(jayess_value *env, jayess_value *size_value) {
    return jayess_http_body_stream_read_chunk(env, size_value, 0);
}

static jayess_value *jayess_std_http_body_stream_read_bytes_method(jayess_value *env, jayess_value *size_value) {
    return jayess_http_body_stream_read_chunk(env, size_value, 1);
}

static jayess_value *jayess_std_http_body_stream_close_method(jayess_value *env) {
    if (env != NULL && env->kind == JAYESS_VALUE_OBJECT && env->as.object_value != NULL) {
        jayess_http_body_stream_mark_ended(env);
        jayess_http_body_stream_close_socket(env);
        jayess_http_body_stream_close_native(env);
    }
    return jayess_value_undefined();
}

static jayess_value *jayess_std_http_body_stream_on_method(jayess_value *env, jayess_value *event, jayess_value *callback) {
    char *event_text = jayess_value_stringify(event);
    if (event_text == NULL || callback == NULL || callback->kind != JAYESS_VALUE_FUNCTION) {
        free(event_text);
        return env != NULL ? env : jayess_value_undefined();
    }
    if (env == NULL || env->kind != JAYESS_VALUE_OBJECT || env->as.object_value == NULL) {
        free(event_text);
        return jayess_value_undefined();
    }
    if (strcmp(event_text, "end") == 0) {
        jayess_std_stream_on(env, "end", callback);
        if (jayess_value_as_bool(jayess_object_get(env->as.object_value, "readableEnded"))) {
            jayess_value_call_one(callback, jayess_value_undefined());
        }
    } else if (strcmp(event_text, "error") == 0) {
        jayess_std_stream_register_error_handler(env, callback);
    } else if (strcmp(event_text, "data") == 0) {
        while (1) {
            jayess_value *chunk = jayess_http_body_stream_read_chunk(env, jayess_value_undefined(), 0);
            if (chunk == NULL || chunk->kind == JAYESS_VALUE_NULL || chunk->kind == JAYESS_VALUE_UNDEFINED) {
                break;
            }
            jayess_value_call_one(callback, chunk);
            if (jayess_has_exception()) {
                break;
            }
        }
    }
    free(event_text);
    return env;
}

static jayess_value *jayess_std_http_body_stream_once_method(jayess_value *env, jayess_value *event, jayess_value *callback) {
    char *event_text = jayess_value_stringify(event);
    if (event_text == NULL || callback == NULL || callback->kind != JAYESS_VALUE_FUNCTION) {
        free(event_text);
        return env != NULL ? env : jayess_value_undefined();
    }
    if (env == NULL || env->kind != JAYESS_VALUE_OBJECT || env->as.object_value == NULL) {
        free(event_text);
        return jayess_value_undefined();
    }
    if (strcmp(event_text, "end") == 0) {
        if (jayess_value_as_bool(jayess_object_get(env->as.object_value, "readableEnded"))) {
            jayess_value_call_one(callback, jayess_value_undefined());
        } else {
            jayess_std_stream_once(env, "end", callback);
        }
    } else if (strcmp(event_text, "error") == 0) {
        jayess_std_stream_register_error_once(env, callback);
    } else if (strcmp(event_text, "data") == 0) {
        jayess_value *chunk = jayess_http_body_stream_read_chunk(env, jayess_value_undefined(), 0);
        if (chunk != NULL && chunk->kind != JAYESS_VALUE_NULL && chunk->kind != JAYESS_VALUE_UNDEFINED) {
            jayess_value_call_one(callback, chunk);
        }
    }
    free(event_text);
    return env;
}

static jayess_value *jayess_std_http_body_stream_pipe_method(jayess_value *env, jayess_value *destination) {
    if (destination == NULL || destination->kind != JAYESS_VALUE_OBJECT) {
        return destination != NULL ? destination : jayess_value_undefined();
    }
    while (1) {
        jayess_value *chunk = jayess_http_body_stream_read_chunk(env, jayess_value_undefined(), 1);
        if (chunk == NULL || chunk->kind == JAYESS_VALUE_NULL || chunk->kind == JAYESS_VALUE_UNDEFINED) {
            break;
        }
        jayess_std_writable_write(destination, chunk);
        if (jayess_has_exception()) {
            break;
        }
    }
    jayess_std_writable_end(destination);
    return destination;
}

static jayess_value *jayess_std_socket_read_method(jayess_value *env, jayess_value *size_value) {
    jayess_socket_handle handle = jayess_std_socket_handle(env);
    int requested = jayess_std_stream_requested_size(size_value, 4095);
    char *buffer;
    int read_count;
    int did_timeout = 0;
    if (handle == JAYESS_INVALID_SOCKET) {
        jayess_std_socket_mark_closed(env);
        return jayess_value_undefined();
    }
    buffer = (char *)malloc((size_t)requested + 1);
    if (buffer == NULL) {
        return jayess_value_undefined();
    }
    if (jayess_std_tls_state(env) != NULL) {
        read_count = jayess_std_tls_read_bytes(env, (unsigned char *)buffer, requested, &did_timeout);
    } else {
        read_count = (int)recv(handle, buffer, requested, 0);
        if (read_count < 0 && (errno == EAGAIN || errno == EWOULDBLOCK)) {
            did_timeout = 1;
        }
    }
    if (read_count <= 0) {
        free(buffer);
        if (read_count < 0) {
            jayess_std_stream_emit_error(env, did_timeout ? "socket read timed out" : "failed to read from socket");
        }
        jayess_std_socket_close_handle(handle);
        jayess_std_socket_set_handle(env, JAYESS_INVALID_SOCKET);
        jayess_std_socket_close_native(env);
        jayess_std_socket_mark_closed(env);
        jayess_std_socket_emit_close(env);
        if (read_count == 0) {
            return jayess_value_null();
        }
        return jayess_value_undefined();
    }
    buffer[read_count] = '\0';
    if (env != NULL && env->kind == JAYESS_VALUE_OBJECT && env->as.object_value != NULL) {
        jayess_value *current = jayess_object_get(env->as.object_value, "bytesRead");
        double total = jayess_value_to_number(current) + (double)read_count;
        jayess_object_set_value(env->as.object_value, "bytesRead", jayess_value_from_number(total));
    }
    {
        jayess_value *result = jayess_value_from_string(buffer);
        free(buffer);
        return result;
    }
}

static jayess_value *jayess_std_socket_read_bytes_method(jayess_value *env, jayess_value *size_value) {
    jayess_socket_handle handle = jayess_std_socket_handle(env);
    int requested = jayess_std_stream_requested_size(size_value, 4095);
    unsigned char *buffer;
    int read_count;
    int did_timeout = 0;
    jayess_value *array_buffer;
    jayess_value *view;
    jayess_array *bytes;
    int i;
    if (handle == JAYESS_INVALID_SOCKET) {
        jayess_std_socket_mark_closed(env);
        return jayess_value_undefined();
    }
    buffer = (unsigned char *)malloc((size_t)requested);
    if (buffer == NULL) {
        return jayess_value_undefined();
    }
    if (jayess_std_tls_state(env) != NULL) {
        read_count = jayess_std_tls_read_bytes(env, buffer, requested, &did_timeout);
    } else {
        read_count = (int)recv(handle, (char *)buffer, requested, 0);
        if (read_count < 0 && (errno == EAGAIN || errno == EWOULDBLOCK)) {
            did_timeout = 1;
        }
    }
    if (read_count <= 0) {
        free(buffer);
        if (read_count < 0) {
            jayess_std_stream_emit_error(env, did_timeout ? "socket read timed out" : "failed to read from socket");
        }
        jayess_std_socket_close_handle(handle);
        jayess_std_socket_set_handle(env, JAYESS_INVALID_SOCKET);
        jayess_std_socket_close_native(env);
        jayess_std_socket_mark_closed(env);
        jayess_std_socket_emit_close(env);
        if (read_count == 0) {
            return jayess_value_null();
        }
        return jayess_value_undefined();
    }
    if (env != NULL && env->kind == JAYESS_VALUE_OBJECT && env->as.object_value != NULL) {
        jayess_value *current = jayess_object_get(env->as.object_value, "bytesRead");
        double total = jayess_value_to_number(current) + (double)read_count;
        jayess_object_set_value(env->as.object_value, "bytesRead", jayess_value_from_number(total));
    }
    array_buffer = jayess_std_array_buffer_new(jayess_value_from_number((double)read_count));
    view = jayess_std_uint8_array_new(array_buffer);
    bytes = jayess_std_bytes_slot(view);
    if (bytes == NULL) {
        free(buffer);
        return jayess_value_undefined();
    }
    for (i = 0; i < read_count; i++) {
        bytes->values[i] = jayess_value_from_number((double)buffer[i]);
    }
    free(buffer);
    return view;
}

static jayess_value *jayess_std_socket_write_method(jayess_value *env, jayess_value *value) {
    jayess_socket_handle handle = jayess_std_socket_handle(env);
    int did_timeout = 0;
    double pending_length = 0.0;
    if (handle == JAYESS_INVALID_SOCKET) {
        jayess_std_socket_mark_closed(env);
        return jayess_value_from_bool(0);
    }
    if (value != NULL && value->kind == JAYESS_VALUE_OBJECT && jayess_std_kind_is(value, "Uint8Array")) {
        jayess_array *bytes = jayess_std_bytes_slot(value);
        int offset = 0;
        int write_ok;
        if (bytes == NULL) {
            return jayess_value_from_bool(0);
        }
        pending_length = (double)bytes->count;
        write_ok = jayess_std_stream_backpressure_note_pending(env, pending_length);
        while (offset < bytes->count) {
            unsigned char chunk[1024];
            int chunk_len = bytes->count - offset;
            int i;
            int sent;
            if (chunk_len > (int)sizeof(chunk)) {
                chunk_len = (int)sizeof(chunk);
            }
            for (i = 0; i < chunk_len; i++) {
                chunk[i] = (unsigned char)((int)jayess_value_to_number(jayess_array_get(bytes, offset + i)) & 255);
            }
            if (jayess_std_tls_state(env) != NULL) {
                sent = jayess_std_tls_write_bytes(env, chunk, chunk_len, &did_timeout);
            } else {
                sent = (int)send(handle, (const char *)chunk, chunk_len, 0);
                if (sent < 0 && (errno == EAGAIN || errno == EWOULDBLOCK)) {
                    did_timeout = 1;
                }
            }
            if (sent <= 0) {
                jayess_std_stream_emit_error(env, did_timeout ? "socket write timed out" : "failed to write to socket");
                return jayess_value_from_bool(0);
            }
            if (env != NULL && env->kind == JAYESS_VALUE_OBJECT && env->as.object_value != NULL) {
                jayess_value *current = jayess_object_get(env->as.object_value, "bytesWritten");
                double total = jayess_value_to_number(current) + (double)sent;
                jayess_object_set_value(env->as.object_value, "bytesWritten", jayess_value_from_number(total));
            }
            offset += sent;
        }
        jayess_std_stream_backpressure_maybe_drain(env, 0.0);
        return jayess_value_from_bool(write_ok);
    }
    {
        char *text = jayess_value_stringify(value);
        size_t length;
        size_t offset = 0;
        int write_ok;
        if (text == NULL) {
            return jayess_value_from_bool(0);
        }
        length = strlen(text);
        pending_length = (double)length;
        write_ok = jayess_std_stream_backpressure_note_pending(env, pending_length);
        while (offset < length) {
            int sent;
            if (jayess_std_tls_state(env) != NULL) {
                sent = jayess_std_tls_write_bytes(env, (const unsigned char *)text + offset, (int)(length - offset), &did_timeout);
            } else {
                sent = (int)send(handle, text + offset, (int)(length - offset), 0);
                if (sent < 0 && (errno == EAGAIN || errno == EWOULDBLOCK)) {
                    did_timeout = 1;
                }
            }
            if (sent <= 0) {
                jayess_std_stream_emit_error(env, did_timeout ? "socket write timed out" : "failed to write to socket");
                free(text);
                return jayess_value_from_bool(0);
            }
            if (env != NULL && env->kind == JAYESS_VALUE_OBJECT && env->as.object_value != NULL) {
                jayess_value *current = jayess_object_get(env->as.object_value, "bytesWritten");
                double total = jayess_value_to_number(current) + (double)sent;
                jayess_object_set_value(env->as.object_value, "bytesWritten", jayess_value_from_number(total));
            }
            offset += (size_t)sent;
        }
        free(text);
        jayess_std_stream_backpressure_maybe_drain(env, 0.0);
        return jayess_value_from_bool(write_ok);
    }
}

static jayess_value *jayess_std_datagram_socket_send_method(jayess_value *env, jayess_value *value, jayess_value *port_value, jayess_value *host_value) {
    jayess_socket_handle handle = jayess_std_socket_handle(env);
    char *host_text = jayess_value_stringify(host_value);
    int port = (int)jayess_value_to_number(port_value);
    char port_text[32];
    struct addrinfo hints;
    struct addrinfo *results = NULL;
    struct addrinfo *entry;
    int status;
    int sent = -1;
    int did_timeout = 0;
    if (handle == JAYESS_INVALID_SOCKET || host_text == NULL || host_text[0] == '\0' || port <= 0) {
        free(host_text);
        return jayess_value_from_bool(0);
    }
    snprintf(port_text, sizeof(port_text), "%d", port);
    memset(&hints, 0, sizeof(hints));
    hints.ai_family = AF_UNSPEC;
    hints.ai_socktype = SOCK_DGRAM;
    status = getaddrinfo(host_text, port_text, &hints, &results);
    if (status != 0 || results == NULL) {
        free(host_text);
        return jayess_value_from_bool(0);
    }
    if (value != NULL && value->kind == JAYESS_VALUE_OBJECT && jayess_std_kind_is(value, "Uint8Array")) {
        jayess_array *bytes = jayess_std_bytes_slot(value);
        unsigned char *buffer;
        int i;
        if (bytes == NULL) {
            freeaddrinfo(results);
            free(host_text);
            return jayess_value_from_bool(0);
        }
        buffer = (unsigned char *)malloc((size_t)bytes->count);
        if (buffer == NULL) {
            freeaddrinfo(results);
            free(host_text);
            return jayess_value_from_bool(0);
        }
        for (i = 0; i < bytes->count; i++) {
            buffer[i] = (unsigned char)((int)jayess_value_to_number(jayess_array_get(bytes, i)) & 255);
        }
        for (entry = results; entry != NULL; entry = entry->ai_next) {
            sent = (int)sendto(handle, (const char *)buffer, bytes->count, 0, entry->ai_addr, (int)entry->ai_addrlen);
#ifdef _WIN32
            if (sent < 0 && WSAGetLastError() == WSAETIMEDOUT) {
                did_timeout = 1;
            }
#else
            if (sent < 0 && (errno == EAGAIN || errno == EWOULDBLOCK)) {
                did_timeout = 1;
            }
#endif
            if (sent >= 0) {
                break;
            }
        }
        free(buffer);
    } else {
        char *text = jayess_value_stringify(value);
        size_t length = text != NULL ? strlen(text) : 0;
        for (entry = results; entry != NULL; entry = entry->ai_next) {
            sent = (int)sendto(handle, text != NULL ? text : "", (int)length, 0, entry->ai_addr, (int)entry->ai_addrlen);
#ifdef _WIN32
            if (sent < 0 && WSAGetLastError() == WSAETIMEDOUT) {
                did_timeout = 1;
            }
#else
            if (sent < 0 && (errno == EAGAIN || errno == EWOULDBLOCK)) {
                did_timeout = 1;
            }
#endif
            if (sent >= 0) {
                break;
            }
        }
        free(text);
    }
    freeaddrinfo(results);
    free(host_text);
    if (sent < 0) {
        jayess_std_stream_emit_error(env, did_timeout ? "datagram send timed out" : "failed to send datagram");
        return jayess_value_from_bool(0);
    }
    if (env != NULL && env->kind == JAYESS_VALUE_OBJECT && env->as.object_value != NULL) {
        jayess_value *current = jayess_object_get(env->as.object_value, "bytesWritten");
        double total = jayess_value_to_number(current) + (double)sent;
        jayess_object_set_value(env->as.object_value, "bytesWritten", jayess_value_from_number(total));
    }
    return jayess_value_from_bool(1);
}

static jayess_value *jayess_std_datagram_socket_receive_method(jayess_value *env, jayess_value *size_value) {
    jayess_socket_handle handle = jayess_std_socket_handle(env);
    int requested = jayess_std_stream_requested_size(size_value, 65535);
    unsigned char *buffer;
    int read_count;
    int did_timeout = 0;
    struct sockaddr_storage from_addr;
    char address[INET6_ADDRSTRLEN];
    int port = 0;
    int family = 0;
    void *addr_ptr = NULL;
    jayess_object *packet;
    jayess_value *bytes_value;
    char *text;
#ifdef _WIN32
    int from_len = sizeof(from_addr);
#else
    socklen_t from_len = sizeof(from_addr);
#endif
    if (handle == JAYESS_INVALID_SOCKET) {
        jayess_std_socket_mark_closed(env);
        return jayess_value_undefined();
    }
    if (requested <= 0) {
        requested = 65535;
    }
    buffer = (unsigned char *)malloc((size_t)requested);
    if (buffer == NULL) {
        return jayess_value_undefined();
    }
    memset(&from_addr, 0, sizeof(from_addr));
    read_count = (int)recvfrom(handle, (char *)buffer, requested, 0, (struct sockaddr *)&from_addr, &from_len);
#ifdef _WIN32
    if (read_count < 0 && WSAGetLastError() == WSAETIMEDOUT) {
        did_timeout = 1;
    }
#else
    if (read_count < 0 && (errno == EAGAIN || errno == EWOULDBLOCK)) {
        did_timeout = 1;
    }
#endif
    if (read_count <= 0) {
        free(buffer);
        if (read_count < 0) {
            jayess_std_stream_emit_error(env, did_timeout ? "datagram receive timed out" : "failed to receive datagram");
        }
        return read_count == 0 ? jayess_value_null() : jayess_value_undefined();
    }
    if (from_addr.ss_family == AF_INET) {
        struct sockaddr_in *ipv4 = (struct sockaddr_in *)&from_addr;
        addr_ptr = &(ipv4->sin_addr);
        port = ntohs(ipv4->sin_port);
        family = 4;
    } else if (from_addr.ss_family == AF_INET6) {
        struct sockaddr_in6 *ipv6 = (struct sockaddr_in6 *)&from_addr;
        addr_ptr = &(ipv6->sin6_addr);
        port = ntohs(ipv6->sin6_port);
        family = 6;
    }
    address[0] = '\0';
    if (addr_ptr == NULL || inet_ntop(from_addr.ss_family, addr_ptr, address, sizeof(address)) == NULL) {
        strcpy(address, "");
    }
    if (env != NULL && env->kind == JAYESS_VALUE_OBJECT && env->as.object_value != NULL) {
        jayess_value *current = jayess_object_get(env->as.object_value, "bytesRead");
        double total = jayess_value_to_number(current) + (double)read_count;
        jayess_object_set_value(env->as.object_value, "bytesRead", jayess_value_from_number(total));
    }
    packet = jayess_object_new();
    bytes_value = jayess_std_uint8_array_from_bytes(buffer, (size_t)read_count);
    text = (char *)malloc((size_t)read_count + 1);
    if (text != NULL) {
        memcpy(text, buffer, (size_t)read_count);
        text[read_count] = '\0';
    }
    free(buffer);
    jayess_object_set_value(packet, "data", jayess_value_from_string(text != NULL ? text : ""));
    jayess_object_set_value(packet, "bytes", bytes_value);
    jayess_object_set_value(packet, "address", jayess_value_from_string(address));
    jayess_object_set_value(packet, "port", jayess_value_from_number((double)port));
    jayess_object_set_value(packet, "family", jayess_value_from_number((double)family));
    free(text);
    return jayess_value_from_object(packet);
}

static jayess_value *jayess_std_datagram_socket_set_broadcast_method(jayess_value *env, jayess_value *enabled) {
    jayess_socket_handle handle = jayess_std_socket_handle(env);
    int flag = jayess_value_as_bool(enabled) ? 1 : 0;
    if (handle == JAYESS_INVALID_SOCKET) {
        jayess_std_socket_mark_closed(env);
        return env != NULL ? env : jayess_value_undefined();
    }
#ifdef _WIN32
    if (setsockopt(handle, SOL_SOCKET, SO_BROADCAST, (const char *)&flag, sizeof(flag)) != 0) {
#else
    if (setsockopt(handle, SOL_SOCKET, SO_BROADCAST, &flag, sizeof(flag)) != 0) {
#endif
        jayess_std_stream_emit_error(env, "failed to configure SO_BROADCAST");
        return env != NULL ? env : jayess_value_undefined();
    }
    if (env != NULL && env->kind == JAYESS_VALUE_OBJECT && env->as.object_value != NULL) {
        jayess_object_set_value(env->as.object_value, "broadcast", jayess_value_from_bool(flag));
    }
    return env != NULL ? env : jayess_value_undefined();
}

static int jayess_std_datagram_ipv4_membership(jayess_socket_handle handle, const char *group_text, const char *interface_text, int join) {
    struct ip_mreq membership;
    memset(&membership, 0, sizeof(membership));
    if (group_text == NULL || group_text[0] == '\0') {
        return 0;
    }
    if (inet_pton(AF_INET, group_text, &membership.imr_multiaddr) != 1) {
        return 0;
    }
    if (interface_text != NULL && interface_text[0] != '\0') {
        if (inet_pton(AF_INET, interface_text, &membership.imr_interface) != 1) {
            return 0;
        }
    } else {
        membership.imr_interface.s_addr = htonl(INADDR_ANY);
    }
#ifdef _WIN32
    return setsockopt(handle, IPPROTO_IP, join ? IP_ADD_MEMBERSHIP : IP_DROP_MEMBERSHIP, (const char *)&membership, sizeof(membership)) == 0;
#else
    return setsockopt(handle, IPPROTO_IP, join ? IP_ADD_MEMBERSHIP : IP_DROP_MEMBERSHIP, &membership, sizeof(membership)) == 0;
#endif
}

static jayess_value *jayess_std_datagram_socket_join_group_method(jayess_value *env, jayess_value *group_value, jayess_value *interface_value) {
    jayess_socket_handle handle = jayess_std_socket_handle(env);
    char *group_text = jayess_value_stringify(group_value);
    char *interface_text = jayess_value_stringify(interface_value);
    if (handle == JAYESS_INVALID_SOCKET) {
        free(group_text);
        free(interface_text);
        jayess_std_socket_mark_closed(env);
        return env != NULL ? env : jayess_value_undefined();
    }
    if (!jayess_std_datagram_ipv4_membership(handle, group_text, interface_text, 1)) {
        jayess_std_stream_emit_error(env, "failed to join multicast group");
    }
    free(group_text);
    free(interface_text);
    return env != NULL ? env : jayess_value_undefined();
}

static jayess_value *jayess_std_datagram_socket_leave_group_method(jayess_value *env, jayess_value *group_value, jayess_value *interface_value) {
    jayess_socket_handle handle = jayess_std_socket_handle(env);
    char *group_text = jayess_value_stringify(group_value);
    char *interface_text = jayess_value_stringify(interface_value);
    if (handle == JAYESS_INVALID_SOCKET) {
        free(group_text);
        free(interface_text);
        jayess_std_socket_mark_closed(env);
        return env != NULL ? env : jayess_value_undefined();
    }
    if (!jayess_std_datagram_ipv4_membership(handle, group_text, interface_text, 0)) {
        jayess_std_stream_emit_error(env, "failed to leave multicast group");
    }
    free(group_text);
    free(interface_text);
    return env != NULL ? env : jayess_value_undefined();
}

static jayess_value *jayess_std_datagram_socket_set_multicast_interface_method(jayess_value *env, jayess_value *interface_value) {
    jayess_socket_handle handle = jayess_std_socket_handle(env);
    char *interface_text = jayess_value_stringify(interface_value);
    struct in_addr interface_addr;
    if (handle == JAYESS_INVALID_SOCKET) {
        free(interface_text);
        jayess_std_socket_mark_closed(env);
        return env != NULL ? env : jayess_value_undefined();
    }
    if (interface_text == NULL || interface_text[0] == '\0' || inet_pton(AF_INET, interface_text, &interface_addr) != 1) {
        free(interface_text);
        jayess_std_stream_emit_error(env, "failed to configure multicast interface");
        return env != NULL ? env : jayess_value_undefined();
    }
#ifdef _WIN32
    if (setsockopt(handle, IPPROTO_IP, IP_MULTICAST_IF, (const char *)&interface_addr, sizeof(interface_addr)) != 0) {
#else
    if (setsockopt(handle, IPPROTO_IP, IP_MULTICAST_IF, &interface_addr, sizeof(interface_addr)) != 0) {
#endif
        free(interface_text);
        jayess_std_stream_emit_error(env, "failed to configure multicast interface");
        return env != NULL ? env : jayess_value_undefined();
    }
    if (env != NULL && env->kind == JAYESS_VALUE_OBJECT && env->as.object_value != NULL) {
        jayess_object_set_value(env->as.object_value, "multicastInterface", jayess_value_from_string(interface_text));
    }
    free(interface_text);
    return env != NULL ? env : jayess_value_undefined();
}

static jayess_value *jayess_std_datagram_socket_set_multicast_loopback_method(jayess_value *env, jayess_value *enabled) {
    jayess_socket_handle handle = jayess_std_socket_handle(env);
#ifdef _WIN32
    BOOL flag = jayess_value_as_bool(enabled) ? TRUE : FALSE;
    if (handle == JAYESS_INVALID_SOCKET) {
        jayess_std_socket_mark_closed(env);
        return env != NULL ? env : jayess_value_undefined();
    }
    if (setsockopt(handle, IPPROTO_IP, IP_MULTICAST_LOOP, (const char *)&flag, sizeof(flag)) != 0) {
#else
    unsigned char flag = jayess_value_as_bool(enabled) ? 1 : 0;
    if (handle == JAYESS_INVALID_SOCKET) {
        jayess_std_socket_mark_closed(env);
        return env != NULL ? env : jayess_value_undefined();
    }
    if (setsockopt(handle, IPPROTO_IP, IP_MULTICAST_LOOP, &flag, sizeof(flag)) != 0) {
#endif
        jayess_std_stream_emit_error(env, "failed to configure multicast loopback");
        return env != NULL ? env : jayess_value_undefined();
    }
    if (env != NULL && env->kind == JAYESS_VALUE_OBJECT && env->as.object_value != NULL) {
        jayess_object_set_value(env->as.object_value, "multicastLoopback", jayess_value_from_bool(flag ? 1 : 0));
    }
    return env != NULL ? env : jayess_value_undefined();
}

static jayess_value *jayess_std_socket_read_async_method(jayess_value *env, jayess_value *size_value) {
    jayess_value *promise = jayess_std_promise_pending();
    jayess_enqueue_socket_read_task(promise, env, size_value);
    return promise;
}

static jayess_value *jayess_std_socket_write_async_method(jayess_value *env, jayess_value *value) {
    jayess_value *promise = jayess_std_promise_pending();
    jayess_enqueue_socket_write_task(promise, env, value);
    return promise;
}

static jayess_value *jayess_std_socket_set_no_delay_method(jayess_value *env, jayess_value *enabled) {
    jayess_socket_handle handle = jayess_std_socket_handle(env);
    int flag = jayess_value_as_bool(enabled) ? 1 : 0;
    if (handle == JAYESS_INVALID_SOCKET) {
        jayess_std_socket_mark_closed(env);
        return env != NULL ? env : jayess_value_undefined();
    }
#ifdef _WIN32
    if (setsockopt(handle, IPPROTO_TCP, TCP_NODELAY, (const char *)&flag, sizeof(flag)) != 0) {
#else
    if (setsockopt(handle, IPPROTO_TCP, TCP_NODELAY, &flag, sizeof(flag)) != 0) {
#endif
        jayess_std_stream_emit_error(env, "failed to configure TCP_NODELAY");
    }
    return env != NULL ? env : jayess_value_undefined();
}

static jayess_value *jayess_std_socket_set_keep_alive_method(jayess_value *env, jayess_value *enabled) {
    jayess_socket_handle handle = jayess_std_socket_handle(env);
    int flag = jayess_value_as_bool(enabled) ? 1 : 0;
    if (handle == JAYESS_INVALID_SOCKET) {
        jayess_std_socket_mark_closed(env);
        return env != NULL ? env : jayess_value_undefined();
    }
#ifdef _WIN32
    if (setsockopt(handle, SOL_SOCKET, SO_KEEPALIVE, (const char *)&flag, sizeof(flag)) != 0) {
#else
    if (setsockopt(handle, SOL_SOCKET, SO_KEEPALIVE, &flag, sizeof(flag)) != 0) {
#endif
        jayess_std_stream_emit_error(env, "failed to configure SO_KEEPALIVE");
    }
    return env != NULL ? env : jayess_value_undefined();
}

static jayess_value *jayess_std_socket_set_timeout_method(jayess_value *env, jayess_value *timeout_ms) {
    jayess_socket_handle handle = jayess_std_socket_handle(env);
    int timeout = (int)jayess_value_to_number(timeout_ms);
    if (timeout < 0) {
        timeout = 0;
    }
    if (handle == JAYESS_INVALID_SOCKET) {
        jayess_std_socket_mark_closed(env);
        return env != NULL ? env : jayess_value_undefined();
    }
    if (!jayess_std_socket_configure_timeout(handle, timeout)) {
        jayess_std_stream_emit_error(env, "failed to configure socket timeout");
        return env != NULL ? env : jayess_value_undefined();
    }
    if (env != NULL && env->kind == JAYESS_VALUE_OBJECT && env->as.object_value != NULL) {
        jayess_object_set_value(env->as.object_value, "timeout", jayess_value_from_number((double)timeout));
    }
    return env != NULL ? env : jayess_value_undefined();
}

static jayess_value *jayess_std_socket_close_method(jayess_value *env) {
    jayess_socket_handle handle = jayess_std_socket_handle(env);
    if (handle != JAYESS_INVALID_SOCKET) {
#ifdef _WIN32
        shutdown(handle, SD_BOTH);
#else
        shutdown(handle, SHUT_RDWR);
#endif
        jayess_std_socket_close_handle(handle);
        jayess_std_socket_set_handle(env, JAYESS_INVALID_SOCKET);
    }
    jayess_std_socket_close_native(env);
    jayess_std_socket_mark_closed(env);
    jayess_std_socket_emit_close(env);
    return env != NULL ? env : jayess_value_undefined();
}

static jayess_value *jayess_std_socket_address_method(jayess_value *env) {
    jayess_object *result;
    if (env == NULL || env->kind != JAYESS_VALUE_OBJECT || env->as.object_value == NULL) {
        return jayess_value_undefined();
    }
    result = jayess_object_new();
    if (result == NULL) {
        return jayess_value_from_object(NULL);
    }
    jayess_object_set_value(result, "address", jayess_object_get(env->as.object_value, "localAddress"));
    jayess_object_set_value(result, "port", jayess_object_get(env->as.object_value, "localPort"));
    jayess_object_set_value(result, "family", jayess_object_get(env->as.object_value, "localFamily"));
    return jayess_value_from_object(result);
}

static jayess_value *jayess_std_socket_remote_method(jayess_value *env) {
    jayess_object *result;
    if (env == NULL || env->kind != JAYESS_VALUE_OBJECT || env->as.object_value == NULL) {
        return jayess_value_undefined();
    }
    result = jayess_object_new();
    if (result == NULL) {
        return jayess_value_from_object(NULL);
    }
    jayess_object_set_value(result, "address", jayess_object_get(env->as.object_value, "remoteAddress"));
    jayess_object_set_value(result, "port", jayess_object_get(env->as.object_value, "remotePort"));
    jayess_object_set_value(result, "family", jayess_object_get(env->as.object_value, "remoteFamily"));
    return jayess_value_from_object(result);
}

static jayess_value *jayess_std_socket_get_peer_certificate_method(jayess_value *env) {
    if (jayess_std_tls_state(env) == NULL) {
        return jayess_value_undefined();
    }
    return jayess_std_tls_peer_certificate(env);
}

static jayess_value *jayess_std_socket_on_method(jayess_value *env, jayess_value *event, jayess_value *callback) {
    char *event_text = jayess_value_stringify(event);
    if (event_text == NULL || callback == NULL || callback->kind != JAYESS_VALUE_FUNCTION) {
        free(event_text);
        return env != NULL ? env : jayess_value_undefined();
    }
    if (env == NULL || env->kind != JAYESS_VALUE_OBJECT || env->as.object_value == NULL) {
        free(event_text);
        return jayess_value_undefined();
    }
    if (strcmp(event_text, "error") == 0) {
        jayess_std_stream_register_error_handler(env, callback);
    } else if (strcmp(event_text, "connect") == 0) {
        jayess_std_stream_on(env, "connect", callback);
        if (jayess_value_as_bool(jayess_object_get(env->as.object_value, "connected"))) {
            jayess_value_call_one(callback, jayess_value_undefined());
        }
    } else if (strcmp(event_text, "close") == 0) {
        jayess_std_stream_on(env, "close", callback);
        if (jayess_value_as_bool(jayess_object_get(env->as.object_value, "closed"))) {
            jayess_value_call_one(callback, jayess_value_undefined());
        }
    } else if (strcmp(event_text, "drain") == 0) {
        jayess_std_stream_on(env, "drain", callback);
    }
    free(event_text);
    return env;
}

static jayess_value *jayess_std_socket_once_method(jayess_value *env, jayess_value *event, jayess_value *callback) {
    char *event_text = jayess_value_stringify(event);
    if (event_text == NULL || callback == NULL || callback->kind != JAYESS_VALUE_FUNCTION) {
        free(event_text);
        return env != NULL ? env : jayess_value_undefined();
    }
    if (env == NULL || env->kind != JAYESS_VALUE_OBJECT || env->as.object_value == NULL) {
        free(event_text);
        return jayess_value_undefined();
    }
    if (strcmp(event_text, "error") == 0) {
        jayess_std_stream_register_error_once(env, callback);
    } else if (strcmp(event_text, "connect") == 0) {
        jayess_std_stream_once(env, "connect", callback);
        if (jayess_value_as_bool(jayess_object_get(env->as.object_value, "connected"))) {
            jayess_std_stream_off(env, "connect", callback);
            jayess_value_call_one(callback, jayess_value_undefined());
        }
    } else if (strcmp(event_text, "close") == 0) {
        if (jayess_value_as_bool(jayess_object_get(env->as.object_value, "closed"))) {
            jayess_value_call_one(callback, jayess_value_undefined());
        } else {
            jayess_std_stream_once(env, "close", callback);
        }
    } else if (strcmp(event_text, "drain") == 0) {
        if (!jayess_std_stream_bool_property(env, "writableNeedDrain")) {
            jayess_value_call_one(callback, jayess_value_undefined());
        } else {
            jayess_std_stream_once(env, "drain", callback);
        }
    }
    free(event_text);
    return env;
}

static jayess_value *jayess_std_server_on_method(jayess_value *env, jayess_value *event, jayess_value *callback) {
    char *event_text = jayess_value_stringify(event);
    if (event_text == NULL || callback == NULL || callback->kind != JAYESS_VALUE_FUNCTION) {
        free(event_text);
        return env != NULL ? env : jayess_value_undefined();
    }
    if (env == NULL || env->kind != JAYESS_VALUE_OBJECT || env->as.object_value == NULL) {
        free(event_text);
        return jayess_value_undefined();
    }
    if (strcmp(event_text, "error") == 0) {
        jayess_std_stream_register_error_handler(env, callback);
    } else if (strcmp(event_text, "close") == 0) {
        jayess_std_stream_on(env, "close", callback);
        if (jayess_value_as_bool(jayess_object_get(env->as.object_value, "closed"))) {
            jayess_value_call_one(callback, jayess_value_undefined());
        }
    } else if (strcmp(event_text, "listening") == 0) {
        jayess_std_stream_on(env, "listening", callback);
        if (jayess_value_as_bool(jayess_object_get(env->as.object_value, "listening"))) {
            jayess_value_call_one(callback, jayess_value_undefined());
        }
    } else if (strcmp(event_text, "connection") == 0) {
        jayess_std_stream_on(env, "connection", callback);
    }
    free(event_text);
    return env;
}

static jayess_value *jayess_std_server_once_method(jayess_value *env, jayess_value *event, jayess_value *callback) {
    char *event_text = jayess_value_stringify(event);
    if (event_text == NULL || callback == NULL || callback->kind != JAYESS_VALUE_FUNCTION) {
        free(event_text);
        return env != NULL ? env : jayess_value_undefined();
    }
    if (env == NULL || env->kind != JAYESS_VALUE_OBJECT || env->as.object_value == NULL) {
        free(event_text);
        return jayess_value_undefined();
    }
    if (strcmp(event_text, "error") == 0) {
        jayess_std_stream_register_error_once(env, callback);
    } else if (strcmp(event_text, "close") == 0) {
        if (jayess_value_as_bool(jayess_object_get(env->as.object_value, "closed"))) {
            jayess_value_call_one(callback, jayess_value_undefined());
        } else {
            jayess_std_stream_once(env, "close", callback);
        }
    } else if (strcmp(event_text, "listening") == 0) {
        jayess_std_stream_once(env, "listening", callback);
        if (jayess_value_as_bool(jayess_object_get(env->as.object_value, "listening"))) {
            jayess_std_stream_off(env, "listening", callback);
            jayess_value_call_one(callback, jayess_value_undefined());
        }
    } else if (strcmp(event_text, "connection") == 0) {
        jayess_std_stream_once(env, "connection", callback);
    }
    free(event_text);
    return env;
}

static jayess_value *jayess_std_server_accept_method(jayess_value *env) {
    jayess_socket_handle handle = jayess_std_socket_handle(env);
    struct sockaddr_storage client_addr;
#ifdef _WIN32
    int client_len = sizeof(client_addr);
#else
    socklen_t client_len = sizeof(client_addr);
#endif
    jayess_socket_handle client_handle;
    char address[INET6_ADDRSTRLEN];
    int port = 0;
    void *addr_ptr = NULL;
    if (handle == JAYESS_INVALID_SOCKET) {
        if (env != NULL && env->kind == JAYESS_VALUE_OBJECT && env->as.object_value != NULL) {
            jayess_object_set_value(env->as.object_value, "closed", jayess_value_from_bool(1));
            jayess_object_set_value(env->as.object_value, "listening", jayess_value_from_bool(0));
        }
        return jayess_value_undefined();
    }
    memset(&client_addr, 0, sizeof(client_addr));
    client_handle = accept(handle, (struct sockaddr *)&client_addr, &client_len);
    if (client_handle == JAYESS_INVALID_SOCKET) {
        jayess_std_stream_emit_error(env, "failed to accept socket connection");
        return jayess_value_undefined();
    }
    address[0] = '\0';
    if (client_addr.ss_family == AF_INET) {
        struct sockaddr_in *ipv4 = (struct sockaddr_in *)&client_addr;
        addr_ptr = &(ipv4->sin_addr);
        port = ntohs(ipv4->sin_port);
    } else if (client_addr.ss_family == AF_INET6) {
        struct sockaddr_in6 *ipv6 = (struct sockaddr_in6 *)&client_addr;
        addr_ptr = &(ipv6->sin6_addr);
        port = ntohs(ipv6->sin6_port);
    }
    if (addr_ptr == NULL || inet_ntop(client_addr.ss_family, addr_ptr, address, sizeof(address)) == NULL) {
        jayess_std_socket_close_handle(client_handle);
        return jayess_value_undefined();
    }
    {
        jayess_value *result = jayess_std_socket_value_from_handle(client_handle, address, port);
        if (env != NULL && env->kind == JAYESS_VALUE_OBJECT && env->as.object_value != NULL) {
            jayess_value *current = jayess_object_get(env->as.object_value, "connectionsAccepted");
            double total = jayess_value_to_number(current) + 1.0;
            jayess_object_set_value(env->as.object_value, "connectionsAccepted", jayess_value_from_number(total));
        }
        jayess_std_socket_set_remote_family(result, client_addr.ss_family == AF_INET6 ? 6 : 4);
        jayess_std_socket_set_local_endpoint(result, client_handle);
        jayess_std_stream_emit(env, "connection", result);
        return result;
    }
}

static jayess_value *jayess_std_server_accept_async_method(jayess_value *env) {
    jayess_value *promise = jayess_std_promise_pending();
    jayess_enqueue_server_accept_task(promise, env);
    return promise;
}

static jayess_value *jayess_std_server_close_method(jayess_value *env) {
    jayess_socket_handle handle = jayess_std_socket_handle(env);
    if (handle != JAYESS_INVALID_SOCKET) {
#ifdef _WIN32
        shutdown(handle, SD_BOTH);
#else
        shutdown(handle, SHUT_RDWR);
#endif
        jayess_std_socket_close_handle(handle);
        jayess_std_socket_set_handle(env, JAYESS_INVALID_SOCKET);
    }
    if (env != NULL && env->kind == JAYESS_VALUE_OBJECT && env->as.object_value != NULL) {
        jayess_object_set_value(env->as.object_value, "closed", jayess_value_from_bool(1));
        jayess_object_set_value(env->as.object_value, "listening", jayess_value_from_bool(0));
    }
    jayess_std_socket_emit_close(env);
    return env != NULL ? env : jayess_value_undefined();
}

static jayess_value *jayess_std_server_set_timeout_method(jayess_value *env, jayess_value *timeout_ms) {
    jayess_socket_handle handle = jayess_std_socket_handle(env);
    int timeout = (int)jayess_value_to_number(timeout_ms);
    if (timeout < 0) {
        timeout = 0;
    }
    if (handle == JAYESS_INVALID_SOCKET) {
        if (env != NULL && env->kind == JAYESS_VALUE_OBJECT && env->as.object_value != NULL) {
            jayess_object_set_value(env->as.object_value, "closed", jayess_value_from_bool(1));
            jayess_object_set_value(env->as.object_value, "listening", jayess_value_from_bool(0));
        }
        return env != NULL ? env : jayess_value_undefined();
    }
#ifdef _WIN32
    {
        DWORD timeout_value = (DWORD)timeout;
        if (setsockopt(handle, SOL_SOCKET, SO_RCVTIMEO, (const char *)&timeout_value, sizeof(timeout_value)) != 0) {
            jayess_std_stream_emit_error(env, "failed to configure server timeout");
            return env != NULL ? env : jayess_value_undefined();
        }
    }
#else
    {
        struct timeval timeout_value;
        timeout_value.tv_sec = timeout / 1000;
        timeout_value.tv_usec = (timeout % 1000) * 1000;
        if (setsockopt(handle, SOL_SOCKET, SO_RCVTIMEO, &timeout_value, sizeof(timeout_value)) != 0) {
            jayess_std_stream_emit_error(env, "failed to configure server timeout");
            return env != NULL ? env : jayess_value_undefined();
        }
    }
#endif
    if (env != NULL && env->kind == JAYESS_VALUE_OBJECT && env->as.object_value != NULL) {
        jayess_object_set_value(env->as.object_value, "timeout", jayess_value_from_number((double)timeout));
    }
    return env != NULL ? env : jayess_value_undefined();
}

static jayess_value *jayess_std_server_address_method(jayess_value *env) {
    jayess_object *result;
    if (env == NULL || env->kind != JAYESS_VALUE_OBJECT || env->as.object_value == NULL) {
        return jayess_value_undefined();
    }
    result = jayess_object_new();
    if (result == NULL) {
        return jayess_value_from_object(NULL);
    }
    jayess_object_set_value(result, "address", jayess_object_get(env->as.object_value, "host"));
    jayess_object_set_value(result, "port", jayess_object_get(env->as.object_value, "port"));
    jayess_object_set_value(result, "family", jayess_object_get(env->as.object_value, "family"));
    return jayess_value_from_object(result);
}

jayess_value *jayess_std_process_exit(jayess_value *code) {
    int exit_code = (int)jayess_value_to_number(code);
    jayess_runtime_shutdown();
    exit(exit_code);
    return jayess_value_undefined();
}

jayess_value *jayess_std_process_argv(void) {
    if (jayess_current_args == NULL) {
        return jayess_value_from_array(jayess_array_new());
    }
    return jayess_value_from_args(jayess_current_args);
}

jayess_value *jayess_std_process_platform(void) {
#ifdef _WIN32
    return jayess_value_from_string("windows");
#elif __APPLE__
    return jayess_value_from_string("darwin");
#else
    return jayess_value_from_string("linux");
#endif
}

jayess_value *jayess_std_process_arch(void) {
#if defined(__aarch64__) || defined(_M_ARM64)
    return jayess_value_from_string("arm64");
#elif defined(__x86_64__) || defined(_M_X64)
    return jayess_value_from_string("x64");
#elif defined(__i386__) || defined(_M_IX86)
    return jayess_value_from_string("x86");
#else
    return jayess_value_from_string("unknown");
#endif
}

jayess_value *jayess_std_process_tmpdir(void) {
#ifdef _WIN32
    char buffer[MAX_PATH];
    DWORD length = GetTempPathA((DWORD)sizeof(buffer), buffer);
    if (length == 0 || length >= (DWORD)sizeof(buffer)) {
        return jayess_value_from_string(".");
    }
    while (length > 0 && (buffer[length - 1] == '\\' || buffer[length - 1] == '/')) {
        buffer[length - 1] = '\0';
        length--;
    }
    return jayess_value_from_string(buffer);
#else
    const char *tmp = getenv("TMPDIR");
    if (tmp == NULL || tmp[0] == '\0') {
        tmp = "/tmp";
    }
    return jayess_value_from_string(tmp);
#endif
}

jayess_value *jayess_std_process_hostname(void) {
#ifdef _WIN32
    char buffer[MAX_COMPUTERNAME_LENGTH + 1];
    DWORD size = (DWORD)sizeof(buffer);
    if (!GetComputerNameA(buffer, &size) || size == 0) {
        return jayess_value_from_string("localhost");
    }
    buffer[size] = '\0';
    return jayess_value_from_string(buffer);
#else
    char buffer[256];
    if (gethostname(buffer, sizeof(buffer)) != 0) {
        return jayess_value_from_string("localhost");
    }
    buffer[sizeof(buffer) - 1] = '\0';
    return jayess_value_from_string(buffer);
#endif
}

double jayess_std_process_uptime(void) {
#ifdef _WIN32
    return (double)GetTickCount64() / 1000.0;
#else
    struct timespec ts;
    if (clock_gettime(CLOCK_MONOTONIC, &ts) != 0) {
        return 0.0;
    }
    return (double)ts.tv_sec + ((double)ts.tv_nsec / 1000000000.0);
#endif
}

double jayess_std_process_hrtime(void) {
#ifdef _WIN32
    LARGE_INTEGER frequency;
    LARGE_INTEGER counter;
    if (!QueryPerformanceFrequency(&frequency) || frequency.QuadPart == 0 || !QueryPerformanceCounter(&counter)) {
        return 0.0;
    }
    return ((double)counter.QuadPart * 1000000000.0) / (double)frequency.QuadPart;
#else
    struct timespec ts;
    if (clock_gettime(CLOCK_MONOTONIC, &ts) != 0) {
        return 0.0;
    }
    return ((double)ts.tv_sec * 1000000000.0) + (double)ts.tv_nsec;
#endif
}

jayess_value *jayess_std_process_cpu_info(void) {
    jayess_object *result = jayess_object_new();
    long count = 1;
#ifdef _WIN32
    SYSTEM_INFO info;
    GetSystemInfo(&info);
    count = (long)info.dwNumberOfProcessors;
#else
    long detected = sysconf(_SC_NPROCESSORS_ONLN);
    if (detected > 0) {
        count = detected;
    }
#endif
    jayess_object_set_value(result, "count", jayess_value_from_number((double)(count > 0 ? count : 1)));
    jayess_object_set_value(result, "arch", jayess_std_process_arch());
    return jayess_value_from_object(result);
}

jayess_value *jayess_std_process_memory_info(void) {
    jayess_object *result = jayess_object_new();
    jayess_object *jayess = jayess_object_new();
    double total = 0;
    double available = 0;
#ifdef _WIN32
    MEMORYSTATUSEX status;
    status.dwLength = sizeof(status);
    if (GlobalMemoryStatusEx(&status)) {
        total = (double)status.ullTotalPhys;
        available = (double)status.ullAvailPhys;
    }
#else
    long pages = sysconf(_SC_PHYS_PAGES);
    long available_pages = sysconf(_SC_AVPHYS_PAGES);
    long page_size = sysconf(_SC_PAGE_SIZE);
    if (pages > 0 && page_size > 0) {
        total = (double)pages * (double)page_size;
    }
    if (available_pages > 0 && page_size > 0) {
        available = (double)available_pages * (double)page_size;
    }
#endif
    jayess_object_set_value(result, "total", jayess_value_from_number(total));
    jayess_object_set_value(result, "available", jayess_value_from_number(available));
    jayess_object_set_value(jayess, "boxedValues", jayess_value_from_number((double)jayess_runtime_accounting_state.boxed_values));
    jayess_object_set_value(jayess, "objects", jayess_value_from_number((double)jayess_runtime_accounting_state.objects));
    jayess_object_set_value(jayess, "objectEntries", jayess_value_from_number((double)jayess_runtime_accounting_state.object_entries));
    jayess_object_set_value(jayess, "arrays", jayess_value_from_number((double)jayess_runtime_accounting_state.arrays));
    jayess_object_set_value(jayess, "arraySlots", jayess_value_from_number((double)jayess_runtime_accounting_state.array_slots));
    jayess_object_set_value(jayess, "functions", jayess_value_from_number((double)jayess_runtime_accounting_state.functions));
    jayess_object_set_value(jayess, "strings", jayess_value_from_number((double)jayess_runtime_accounting_state.strings));
    jayess_object_set_value(jayess, "bigints", jayess_value_from_number((double)jayess_runtime_accounting_state.bigints));
    jayess_object_set_value(jayess, "symbols", jayess_value_from_number((double)jayess_runtime_accounting_state.symbols));
    jayess_object_set_value(jayess, "nativeHandleWrappers", jayess_value_from_number((double)jayess_runtime_accounting_state.native_handle_wrappers));
    jayess_object_set_value(result, "jayess", jayess_value_from_object(jayess));
    return jayess_value_from_object(result);
}

jayess_value *jayess_std_process_user_info(void) {
    jayess_object *result = jayess_object_new();
    const char *username = NULL;
    const char *home = NULL;
#ifdef _WIN32
    char username_buffer[256];
    DWORD username_size = (DWORD)sizeof(username_buffer);
    if (GetUserNameA(username_buffer, &username_size) && username_size > 0) {
        username = username_buffer;
    } else {
        username = getenv("USERNAME");
    }
    home = getenv("USERPROFILE");
#else
    struct passwd *pwd = getpwuid(getuid());
    if (pwd != NULL && pwd->pw_name != NULL && pwd->pw_name[0] != '\0') {
        username = pwd->pw_name;
    } else {
        username = getenv("USER");
    }
    if (pwd != NULL && pwd->pw_dir != NULL && pwd->pw_dir[0] != '\0') {
        home = pwd->pw_dir;
    } else {
        home = getenv("HOME");
    }
#endif
    if (username == NULL || username[0] == '\0') {
        username = "unknown";
    }
    if (home == NULL || home[0] == '\0') {
        home = "";
    }
    jayess_object_set_value(result, "username", jayess_value_from_string(username));
    jayess_object_set_value(result, "home", jayess_value_from_string(home));
    return jayess_value_from_object(result);
}

jayess_value *jayess_std_process_thread_pool_size(void) {
    return jayess_value_from_number((double)JAYESS_IO_WORKER_COUNT);
}

jayess_value *jayess_std_process_on_signal(jayess_value *signal, jayess_value *callback) {
    char *signal_name = jayess_value_stringify(signal);
    int signal_number;
    jayess_value *bus;
    if (signal_name == NULL || callback == NULL || callback->kind != JAYESS_VALUE_FUNCTION) {
        free(signal_name);
        return jayess_value_undefined();
    }
    signal_number = jayess_std_child_process_signal_number(signal_name);
    if (signal_number == 0) {
        free(signal_name);
        jayess_throw(jayess_type_error_value("unsupported process signal"));
        return jayess_value_undefined();
    }
    if (!jayess_std_process_install_signal(signal_number)) {
        free(signal_name);
        jayess_throw(jayess_type_error_value("failed to install process signal handler"));
        return jayess_value_undefined();
    }
    bus = jayess_std_process_signal_bus_value();
    jayess_std_stream_on(bus, jayess_std_process_signal_name(signal_number), callback);
    free(signal_name);
    return bus;
}

jayess_value *jayess_std_process_once_signal(jayess_value *signal, jayess_value *callback) {
    char *signal_name = jayess_value_stringify(signal);
    int signal_number;
    jayess_value *bus;
    if (signal_name == NULL || callback == NULL || callback->kind != JAYESS_VALUE_FUNCTION) {
        free(signal_name);
        return jayess_value_undefined();
    }
    signal_number = jayess_std_child_process_signal_number(signal_name);
    if (signal_number == 0) {
        free(signal_name);
        jayess_throw(jayess_type_error_value("unsupported process signal"));
        return jayess_value_undefined();
    }
    if (!jayess_std_process_install_signal(signal_number)) {
        free(signal_name);
        jayess_throw(jayess_type_error_value("failed to install process signal handler"));
        return jayess_value_undefined();
    }
    bus = jayess_std_process_signal_bus_value();
    jayess_std_stream_once(bus, jayess_std_process_signal_name(signal_number), callback);
    free(signal_name);
    return bus;
}

jayess_value *jayess_std_process_off_signal(jayess_value *signal, jayess_value *callback) {
    char *signal_name = jayess_value_stringify(signal);
    int signal_number;
    jayess_value *bus;
    if (signal_name == NULL) {
        free(signal_name);
        return jayess_value_undefined();
    }
    signal_number = jayess_std_child_process_signal_number(signal_name);
    if (signal_number == 0) {
        free(signal_name);
        jayess_throw(jayess_type_error_value("unsupported process signal"));
        return jayess_value_undefined();
    }
    bus = jayess_std_process_signal_bus_value();
    jayess_std_stream_off(bus, jayess_std_process_signal_name(signal_number), callback);
    free(signal_name);
    return bus;
}

jayess_value *jayess_std_process_raise(jayess_value *signal) {
    char *signal_name = jayess_value_stringify(signal);
    int signal_number;
    if (signal_name == NULL) {
        free(signal_name);
        return jayess_value_from_bool(0);
    }
    signal_number = jayess_std_child_process_signal_number(signal_name);
    free(signal_name);
    if (signal_number == 0) {
        jayess_throw(jayess_type_error_value("unsupported process signal"));
        return jayess_value_undefined();
    }
    if (!jayess_std_process_install_signal(signal_number)) {
        jayess_throw(jayess_type_error_value("failed to install process signal handler"));
        return jayess_value_undefined();
    }
    if (raise(signal_number) != 0) {
        return jayess_value_from_bool(0);
    }
    jayess_runtime_dispatch_pending_signals();
    return jayess_value_from_bool(1);
}

jayess_value *jayess_std_tls_is_available(void) {
    return jayess_value_from_bool(1);
}

jayess_value *jayess_std_tls_backend(void) {
#ifdef _WIN32
    return jayess_value_from_string("schannel");
#else
    return jayess_value_from_string("openssl");
#endif
}

jayess_value *jayess_std_tls_connect(jayess_value *options) {
    return jayess_std_tls_connect_socket(options);
}

jayess_value *jayess_std_https_is_available(void) {
    return jayess_std_tls_is_available();
}

jayess_value *jayess_std_https_backend(void) {
    return jayess_std_tls_backend();
}

static char *jayess_shell_quote(const char *value) {
    size_t len;
    size_t out_len = 2;
    size_t i;
    size_t j = 0;
    char *out;
    if (value == NULL) {
        value = "";
    }
    len = strlen(value);
    for (i = 0; i < len; i++) {
        out_len += (value[i] == '"' || value[i] == '\\') ? 2 : 1;
    }
    out = (char *)malloc(out_len + 1);
    if (out == NULL) {
        return NULL;
    }
    out[j++] = '"';
    for (i = 0; i < len; i++) {
        if (value[i] == '"' || value[i] == '\\') {
            out[j++] = '\\';
        }
        out[j++] = value[i];
    }
    out[j++] = '"';
    out[j] = '\0';
    return out;
}

static char *jayess_compile_flag(const char *name, const char *value) {
    size_t len;
    char *out;
    if (name == NULL || value == NULL || value[0] == '\0') {
        return NULL;
    }
    len = strlen(name) + strlen(value) + 1;
    out = (char *)malloc(len + 1);
    if (out == NULL) {
        return NULL;
    }
    sprintf(out, "%s=%s", name, value);
    return out;
}

#ifdef _WIN32
static int jayess_spawn_compiler(const char *compiler, const char *emit_arg, const char *target_arg, const char *warnings_arg, const char *output_path, const char *source_path, const char *stdout_path, const char *stderr_path) {
    char *quoted_compiler = jayess_shell_quote(compiler);
    char *quoted_output = jayess_shell_quote(output_path);
    char *quoted_source = jayess_shell_quote(source_path);
    char *command;
    STARTUPINFOA startup;
    PROCESS_INFORMATION process;
    SECURITY_ATTRIBUTES security;
    HANDLE stdout_handle;
    HANDLE stderr_handle;
    DWORD exit_code = 1;
    if (quoted_compiler == NULL || quoted_output == NULL || quoted_source == NULL) {
        free(quoted_compiler);
        free(quoted_output);
        free(quoted_source);
        return -1;
    }
    command = (char *)malloc(strlen(quoted_compiler) + strlen(emit_arg) + (target_arg != NULL ? strlen(target_arg) + 1 : 0) + (warnings_arg != NULL ? strlen(warnings_arg) + 1 : 0) + strlen(quoted_output) + strlen(quoted_source) + 16);
    if (command == NULL) {
        free(quoted_compiler);
        free(quoted_output);
        free(quoted_source);
        return -1;
    }
    sprintf(command, "%s %s%s%s%s%s -o %s %s",
            quoted_compiler,
            emit_arg,
            target_arg != NULL ? " " : "",
            target_arg != NULL ? target_arg : "",
            warnings_arg != NULL ? " " : "",
            warnings_arg != NULL ? warnings_arg : "",
            quoted_output,
            quoted_source);
    security.nLength = sizeof(security);
    security.lpSecurityDescriptor = NULL;
    security.bInheritHandle = TRUE;
    stdout_handle = CreateFileA(stdout_path, GENERIC_WRITE, FILE_SHARE_READ, &security, CREATE_ALWAYS, FILE_ATTRIBUTE_NORMAL, NULL);
    stderr_handle = CreateFileA(stderr_path, GENERIC_WRITE, FILE_SHARE_READ, &security, CREATE_ALWAYS, FILE_ATTRIBUTE_NORMAL, NULL);
    if (stdout_handle == INVALID_HANDLE_VALUE || stderr_handle == INVALID_HANDLE_VALUE) {
        if (stdout_handle != INVALID_HANDLE_VALUE) {
            CloseHandle(stdout_handle);
        }
        if (stderr_handle != INVALID_HANDLE_VALUE) {
            CloseHandle(stderr_handle);
        }
        free(quoted_compiler);
        free(quoted_output);
        free(quoted_source);
        free(command);
        return -1;
    }
    ZeroMemory(&startup, sizeof(startup));
    ZeroMemory(&process, sizeof(process));
    startup.cb = sizeof(startup);
    startup.dwFlags = STARTF_USESTDHANDLES;
    startup.hStdOutput = stdout_handle;
    startup.hStdError = stderr_handle;
    startup.hStdInput = GetStdHandle(STD_INPUT_HANDLE);
    if (!CreateProcessA(NULL, command, NULL, NULL, TRUE, 0, NULL, NULL, &startup, &process)) {
        CloseHandle(stdout_handle);
        CloseHandle(stderr_handle);
        free(quoted_compiler);
        free(quoted_output);
        free(quoted_source);
        free(command);
        return -1;
    }
    WaitForSingleObject(process.hProcess, INFINITE);
    GetExitCodeProcess(process.hProcess, &exit_code);
    CloseHandle(process.hProcess);
    CloseHandle(process.hThread);
    CloseHandle(stdout_handle);
    CloseHandle(stderr_handle);
    free(quoted_compiler);
    free(quoted_output);
    free(quoted_source);
    free(command);
    return (int)exit_code;
}
#else
static int jayess_spawn_compiler(const char *compiler, const char *emit_arg, const char *target_arg, const char *warnings_arg, const char *output_path, const char *source_path, const char *stdout_path, const char *stderr_path) {
    int stdout_fd = open(stdout_path, O_CREAT | O_TRUNC | O_WRONLY, 0600);
    int stderr_fd = open(stderr_path, O_CREAT | O_TRUNC | O_WRONLY, 0600);
    int status = 1;
    pid_t pid;
    if (stdout_fd < 0 || stderr_fd < 0) {
        if (stdout_fd >= 0) {
            close(stdout_fd);
        }
        if (stderr_fd >= 0) {
            close(stderr_fd);
        }
        return -1;
    }
    pid = fork();
    if (pid < 0) {
        close(stdout_fd);
        close(stderr_fd);
        return -1;
    }
    if (pid == 0) {
        char *argv[9];
        int argc = 0;
        dup2(stdout_fd, STDOUT_FILENO);
        dup2(stderr_fd, STDERR_FILENO);
        close(stdout_fd);
        close(stderr_fd);
        argv[argc++] = (char *)compiler;
        argv[argc++] = (char *)emit_arg;
        if (target_arg != NULL) {
            argv[argc++] = (char *)target_arg;
        }
        if (warnings_arg != NULL) {
            argv[argc++] = (char *)warnings_arg;
        }
        argv[argc++] = "-o";
        argv[argc++] = (char *)output_path;
        argv[argc++] = (char *)source_path;
        argv[argc] = NULL;
        execvp(compiler, argv);
        _exit(127);
    }
    close(stdout_fd);
    close(stderr_fd);
    if (waitpid(pid, &status, 0) < 0) {
        return -1;
    }
    if (WIFEXITED(status)) {
        return WEXITSTATUS(status);
    }
    return -1;
}
#endif

const char *jayess_temp_dir(void) {
#ifdef _WIN32
    const char *tmp = getenv("TEMP");
    if (tmp == NULL || tmp[0] == '\0') {
        tmp = getenv("TMP");
    }
    return (tmp != NULL && tmp[0] != '\0') ? tmp : ".";
#else
    const char *tmp = getenv("TMPDIR");
    return (tmp != NULL && tmp[0] != '\0') ? tmp : "/tmp";
#endif
}

char *jayess_read_text_file_or_empty(const char *path) {
    FILE *file;
    long size;
    char *text;
    size_t read_count;
    if (path == NULL) {
        return jayess_strdup("");
    }
    file = fopen(path, "rb");
    if (file == NULL) {
        return jayess_strdup("");
    }
    if (fseek(file, 0, SEEK_END) != 0) {
        fclose(file);
        return jayess_strdup("");
    }
    size = ftell(file);
    if (size < 0) {
        fclose(file);
        return jayess_strdup("");
    }
    rewind(file);
    text = (char *)malloc((size_t)size + 1);
    if (text == NULL) {
        fclose(file);
        return jayess_strdup("");
    }
    read_count = fread(text, 1, (size_t)size, file);
    text[read_count] = '\0';
    fclose(file);
    return text;
}

static int jayess_compile_is_safe_flag_value(const char *value) {
    size_t i;
    if (value == NULL || value[0] == '\0') {
        return 1;
    }
    for (i = 0; value[i] != '\0'; i++) {
        unsigned char ch = (unsigned char)value[i];
        if (!(isalnum(ch) || ch == '-' || ch == '_' || ch == '.')) {
            return 0;
        }
    }
    return 1;
}

static int jayess_compile_emit_is_valid(const char *value) {
    return value == NULL || value[0] == '\0' || strcmp(value, "exe") == 0 || strcmp(value, "llvm") == 0;
}

static int jayess_compile_warnings_is_valid(const char *value) {
    return value == NULL || value[0] == '\0' || strcmp(value, "default") == 0 || strcmp(value, "none") == 0 || strcmp(value, "error") == 0;
}

static jayess_value *jayess_compile_invalid_options_result(char *source_text, char *output_text, char *target_text, char *emit_text, char *warnings_text, const char *message) {
    jayess_object *result = jayess_object_new();
    jayess_object_set_value(result, "ok", jayess_value_from_bool(0));
    jayess_object_set_value(result, "output", output_text != NULL ? jayess_value_from_string(output_text) : jayess_value_undefined());
    jayess_object_set_value(result, "status", jayess_value_from_number(-1));
    jayess_object_set_value(result, "stdout", jayess_value_from_string(""));
    jayess_object_set_value(result, "stderr", jayess_value_from_string(""));
    jayess_object_set_value(result, "error", jayess_value_from_string(message != NULL ? message : "invalid compile options"));
    free(source_text);
    free(output_text);
    free(target_text);
    free(emit_text);
    free(warnings_text);
    return jayess_value_from_object(result);
}

static jayess_value *jayess_std_compile_impl(jayess_value *input, jayess_value *options, int input_is_path) {
    char *source_text = input_is_path ? NULL : jayess_value_stringify(input);
    char *input_path_text = input_is_path ? jayess_value_stringify(input) : NULL;
    char *output_text = NULL;
    char *target_text = NULL;
    char *emit_text = NULL;
    char *warnings_text = NULL;
    const char *compiler = getenv("JAYESS_COMPILER");
    const char *tmp_dir = jayess_temp_dir();
    char temp_source_path[4096];
    char default_output[4096];
    char stdout_path[4096];
    char stderr_path[4096];
    char *emit_arg = NULL;
    char *target_arg = NULL;
    char *warnings_arg = NULL;
    char *stdout_text = NULL;
    char *stderr_text = NULL;
    FILE *file;
    int status;
    jayess_object *result = jayess_object_new();
    long stamp = (long)time(NULL);
#ifdef _WIN32
    const char *exe_suffix = ".exe";
    const char sep = '\\';
#else
    const char *exe_suffix = "";
    const char sep = '/';
#endif
    if (compiler == NULL || compiler[0] == '\0') {
        compiler = "jayess";
    }
    if (options != NULL && !jayess_value_is_nullish(options)) {
        if (options->kind == JAYESS_VALUE_OBJECT && options->as.object_value != NULL) {
            output_text = jayess_compile_option_string(options, "output");
            target_text = jayess_compile_option_string(options, "target");
            emit_text = jayess_compile_option_string(options, "emit");
            warnings_text = jayess_compile_option_string(options, "warnings");
        } else {
            output_text = jayess_value_stringify(options);
        }
    }
    if (input_is_path && (input_path_text == NULL || input_path_text[0] == '\0')) {
        return jayess_compile_invalid_options_result(source_text, output_text, target_text, emit_text, warnings_text, "compileFile expects a non-empty input path");
    }
    if (!input_is_path && source_text == NULL) {
        source_text = jayess_strdup("");
    }
    if (!jayess_compile_emit_is_valid(emit_text)) {
        return jayess_compile_invalid_options_result(source_text, output_text, target_text, emit_text, warnings_text, "compile option emit must be \"exe\" or \"llvm\"");
    }
    if (!jayess_compile_warnings_is_valid(warnings_text)) {
        return jayess_compile_invalid_options_result(source_text, output_text, target_text, emit_text, warnings_text, "compile option warnings must be \"default\", \"none\", or \"error\"");
    }
    if (!jayess_compile_is_safe_flag_value(target_text)) {
        return jayess_compile_invalid_options_result(source_text, output_text, target_text, emit_text, warnings_text, "compile option target contains unsupported characters");
    }
    if (!jayess_compile_is_safe_flag_value(emit_text) || !jayess_compile_is_safe_flag_value(warnings_text)) {
        return jayess_compile_invalid_options_result(source_text, output_text, target_text, emit_text, warnings_text, "compile options contain unsupported characters");
    }
    snprintf(temp_source_path, sizeof(temp_source_path), "%s%cjayess-runtime-%ld-%d.js", tmp_dir, sep, stamp, rand());
    snprintf(stdout_path, sizeof(stdout_path), "%s%cjayess-runtime-%ld-%d.stdout", tmp_dir, sep, stamp, rand());
    snprintf(stderr_path, sizeof(stderr_path), "%s%cjayess-runtime-%ld-%d.stderr", tmp_dir, sep, stamp, rand());
    if (output_text == NULL || output_text[0] == '\0') {
        snprintf(default_output, sizeof(default_output), "%s%cjayess-runtime-%ld-%d%s", tmp_dir, sep, stamp, rand(), exe_suffix);
        if (output_text != NULL) {
            free(output_text);
        }
        output_text = jayess_strdup(default_output);
    }
    if (!input_is_path) {
        file = fopen(temp_source_path, "wb");
        if (file == NULL) {
            jayess_object_set_value(result, "ok", jayess_value_from_bool(0));
            jayess_object_set_value(result, "output", output_text != NULL ? jayess_value_from_string(output_text) : jayess_value_undefined());
            jayess_object_set_value(result, "stdout", jayess_value_from_string(""));
            jayess_object_set_value(result, "stderr", jayess_value_from_string(""));
            jayess_object_set_value(result, "error", jayess_value_from_string("failed to create temporary source file"));
            free(source_text);
            free(input_path_text);
            free(output_text);
            free(target_text);
            free(emit_text);
            free(warnings_text);
            return jayess_value_from_object(result);
        }
        fwrite(source_text, 1, strlen(source_text), file);
        fclose(file);
    }
    emit_arg = jayess_compile_flag("--emit", emit_text != NULL && emit_text[0] != '\0' ? emit_text : "exe");
    target_arg = target_text != NULL && target_text[0] != '\0' ? jayess_compile_flag("--target", target_text) : NULL;
    warnings_arg = warnings_text != NULL && warnings_text[0] != '\0' ? jayess_compile_flag("--warnings", warnings_text) : NULL;
    if (emit_arg == NULL || (target_text != NULL && target_text[0] != '\0' && target_arg == NULL) || (warnings_text != NULL && warnings_text[0] != '\0' && warnings_arg == NULL)) {
        jayess_object_set_value(result, "ok", jayess_value_from_bool(0));
        jayess_object_set_value(result, "output", output_text != NULL ? jayess_value_from_string(output_text) : jayess_value_undefined());
        jayess_object_set_value(result, "stdout", jayess_value_from_string(""));
        jayess_object_set_value(result, "stderr", jayess_value_from_string(""));
        jayess_object_set_value(result, "error", jayess_value_from_string("failed to build compiler command"));
        if (!input_is_path) {
            remove(temp_source_path);
        }
        free(source_text);
        free(input_path_text);
        free(output_text);
        free(target_text);
        free(emit_text);
        free(warnings_text);
        free(emit_arg);
        free(target_arg);
        free(warnings_arg);
        return jayess_value_from_object(result);
    }
    status = jayess_spawn_compiler(compiler, emit_arg, target_arg, warnings_arg, output_text, input_is_path ? input_path_text : temp_source_path, stdout_path, stderr_path);
    stdout_text = jayess_read_text_file_or_empty(stdout_path);
    stderr_text = jayess_read_text_file_or_empty(stderr_path);
    if (!input_is_path) {
        remove(temp_source_path);
    }
    remove(stdout_path);
    remove(stderr_path);
    jayess_object_set_value(result, "ok", jayess_value_from_bool(status == 0));
    jayess_object_set_value(result, "output", jayess_value_from_string(output_text));
    jayess_object_set_value(result, "status", jayess_value_from_number((double)status));
    jayess_object_set_value(result, "stdout", jayess_value_from_string(stdout_text != NULL ? stdout_text : ""));
    jayess_object_set_value(result, "stderr", jayess_value_from_string(stderr_text != NULL ? stderr_text : ""));
    jayess_object_set_value(result, "error", status == 0 ? jayess_value_undefined() : jayess_value_from_string((stderr_text != NULL && stderr_text[0] != '\0') ? stderr_text : "compiler command failed"));
    free(source_text);
    free(input_path_text);
    free(output_text);
    free(target_text);
    free(emit_text);
    free(warnings_text);
    free(emit_arg);
    free(target_arg);
    free(warnings_arg);
    free(stdout_text);
    free(stderr_text);
    return jayess_value_from_object(result);
}

jayess_value *jayess_std_compile(jayess_value *source, jayess_value *options) {
    return jayess_std_compile_impl(source, options, 0);
}

jayess_value *jayess_std_compile_file(jayess_value *input_path, jayess_value *options) {
    return jayess_std_compile_impl(input_path, options, 1);
}

jayess_value *jayess_std_crypto_random_bytes(jayess_value *length_value) {
    int length = (int)jayess_value_to_number(length_value);
    unsigned char *buffer;
    jayess_value *result;
    if (length <= 0) {
        return jayess_std_uint8_array_new(jayess_value_from_number(0));
    }
    buffer = (unsigned char *)malloc((size_t)length);
    if (buffer == NULL) {
        return jayess_std_uint8_array_new(jayess_value_from_number(0));
    }
#ifdef _WIN32
    if (BCryptGenRandom(NULL, buffer, (ULONG)length, BCRYPT_USE_SYSTEM_PREFERRED_RNG) < 0) {
        free(buffer);
        return jayess_std_uint8_array_new(jayess_value_from_number(0));
    }
#else
    if (RAND_bytes(buffer, length) != 1) {
        free(buffer);
        return jayess_std_uint8_array_new(jayess_value_from_number(0));
    }
#endif
    result = jayess_std_uint8_array_from_bytes(buffer, (size_t)length);
    free(buffer);
    return result;
}

jayess_value *jayess_std_crypto_hash(jayess_value *algorithm, jayess_value *value) {
    unsigned char *input = NULL;
    size_t input_length = 0;
    char *algorithm_text = jayess_value_stringify(algorithm);
    char *hex = NULL;
    jayess_value *result;
    if (algorithm_text == NULL || !jayess_std_crypto_copy_bytes(value, &input, &input_length)) {
        free(algorithm_text);
        free(input);
        return jayess_value_undefined();
    }
#ifdef _WIN32
    {
        LPCWSTR algorithm_id = jayess_std_crypto_algorithm_id(algorithm_text);
        BCRYPT_ALG_HANDLE provider = NULL;
        BCRYPT_HASH_HANDLE hash = NULL;
        DWORD object_length = 0;
        DWORD hash_length = 0;
        DWORD bytes_written = 0;
        PUCHAR object_buffer = NULL;
        PUCHAR digest = NULL;
        if (algorithm_id == NULL ||
            BCryptOpenAlgorithmProvider(&provider, algorithm_id, NULL, 0) < 0 ||
            BCryptGetProperty(provider, BCRYPT_OBJECT_LENGTH, (PUCHAR)&object_length, sizeof(object_length), &bytes_written, 0) < 0 ||
            BCryptGetProperty(provider, BCRYPT_HASH_LENGTH, (PUCHAR)&hash_length, sizeof(hash_length), &bytes_written, 0) < 0) {
            if (provider != NULL) {
                BCryptCloseAlgorithmProvider(provider, 0);
            }
            free(algorithm_text);
            free(input);
            return jayess_value_undefined();
        }
        object_buffer = (PUCHAR)malloc(object_length > 0 ? object_length : 1);
        digest = (PUCHAR)malloc(hash_length > 0 ? hash_length : 1);
        if (object_buffer == NULL || digest == NULL ||
            BCryptCreateHash(provider, &hash, object_buffer, object_length, NULL, 0, 0) < 0 ||
            BCryptHashData(hash, input, (ULONG)input_length, 0) < 0 ||
            BCryptFinishHash(hash, digest, hash_length, 0) < 0) {
            if (hash != NULL) {
                BCryptDestroyHash(hash);
            }
            if (provider != NULL) {
                BCryptCloseAlgorithmProvider(provider, 0);
            }
            free(object_buffer);
            free(digest);
            free(algorithm_text);
            free(input);
            return jayess_value_undefined();
        }
        hex = jayess_std_crypto_hex_encode(digest, (size_t)hash_length);
        BCryptDestroyHash(hash);
        BCryptCloseAlgorithmProvider(provider, 0);
        free(object_buffer);
        free(digest);
    }
#else
    {
        const EVP_MD *md = EVP_get_digestbyname(algorithm_text);
        EVP_MD_CTX *ctx = NULL;
        unsigned char digest[EVP_MAX_MD_SIZE];
        unsigned int digest_length = 0;
        if (md == NULL) {
            free(algorithm_text);
            free(input);
            return jayess_value_undefined();
        }
        ctx = EVP_MD_CTX_new();
        if (ctx == NULL ||
            EVP_DigestInit_ex(ctx, md, NULL) != 1 ||
            EVP_DigestUpdate(ctx, input, input_length) != 1 ||
            EVP_DigestFinal_ex(ctx, digest, &digest_length) != 1) {
            if (ctx != NULL) {
                EVP_MD_CTX_free(ctx);
            }
            free(algorithm_text);
            free(input);
            return jayess_value_undefined();
        }
        hex = jayess_std_crypto_hex_encode(digest, (size_t)digest_length);
        EVP_MD_CTX_free(ctx);
    }
#endif
    free(algorithm_text);
    free(input);
    if (hex == NULL) {
        return jayess_value_undefined();
    }
    result = jayess_value_from_string(hex);
    free(hex);
    return result;
}

jayess_value *jayess_std_crypto_hmac(jayess_value *algorithm, jayess_value *key, jayess_value *value) {
    unsigned char *key_bytes = NULL;
    unsigned char *value_bytes = NULL;
    size_t key_length = 0;
    size_t value_length = 0;
    char *algorithm_text = jayess_value_stringify(algorithm);
    char *hex = NULL;
    jayess_value *result;
    if (algorithm_text == NULL ||
        !jayess_std_crypto_copy_bytes(key, &key_bytes, &key_length) ||
        !jayess_std_crypto_copy_bytes(value, &value_bytes, &value_length)) {
        free(algorithm_text);
        free(key_bytes);
        free(value_bytes);
        return jayess_value_undefined();
    }
#ifdef _WIN32
    {
        LPCWSTR algorithm_id = jayess_std_crypto_algorithm_id(algorithm_text);
        BCRYPT_ALG_HANDLE provider = NULL;
        BCRYPT_HASH_HANDLE hash = NULL;
        DWORD object_length = 0;
        DWORD hash_length = 0;
        DWORD bytes_written = 0;
        PUCHAR object_buffer = NULL;
        PUCHAR digest = NULL;
        if (algorithm_id == NULL ||
            BCryptOpenAlgorithmProvider(&provider, algorithm_id, NULL, BCRYPT_ALG_HANDLE_HMAC_FLAG) < 0 ||
            BCryptGetProperty(provider, BCRYPT_OBJECT_LENGTH, (PUCHAR)&object_length, sizeof(object_length), &bytes_written, 0) < 0 ||
            BCryptGetProperty(provider, BCRYPT_HASH_LENGTH, (PUCHAR)&hash_length, sizeof(hash_length), &bytes_written, 0) < 0) {
            if (provider != NULL) {
                BCryptCloseAlgorithmProvider(provider, 0);
            }
            free(algorithm_text);
            free(key_bytes);
            free(value_bytes);
            return jayess_value_undefined();
        }
        object_buffer = (PUCHAR)malloc(object_length > 0 ? object_length : 1);
        digest = (PUCHAR)malloc(hash_length > 0 ? hash_length : 1);
        if (object_buffer == NULL || digest == NULL ||
            BCryptCreateHash(provider, &hash, object_buffer, object_length, key_bytes, (ULONG)key_length, 0) < 0 ||
            BCryptHashData(hash, value_bytes, (ULONG)value_length, 0) < 0 ||
            BCryptFinishHash(hash, digest, hash_length, 0) < 0) {
            if (hash != NULL) {
                BCryptDestroyHash(hash);
            }
            if (provider != NULL) {
                BCryptCloseAlgorithmProvider(provider, 0);
            }
            free(object_buffer);
            free(digest);
            free(algorithm_text);
            free(key_bytes);
            free(value_bytes);
            return jayess_value_undefined();
        }
        hex = jayess_std_crypto_hex_encode(digest, (size_t)hash_length);
        BCryptDestroyHash(hash);
        BCryptCloseAlgorithmProvider(provider, 0);
        free(object_buffer);
        free(digest);
    }
#else
    {
        const EVP_MD *md = EVP_get_digestbyname(algorithm_text);
        unsigned char digest[EVP_MAX_MD_SIZE];
        unsigned int digest_length = 0;
        if (md == NULL || HMAC(md, key_bytes, (int)key_length, value_bytes, value_length, digest, &digest_length) == NULL) {
            free(algorithm_text);
            free(key_bytes);
            free(value_bytes);
            return jayess_value_undefined();
        }
        hex = jayess_std_crypto_hex_encode(digest, (size_t)digest_length);
    }
#endif
    free(algorithm_text);
    free(key_bytes);
    free(value_bytes);
    if (hex == NULL) {
        return jayess_value_undefined();
    }
    result = jayess_value_from_string(hex);
    free(hex);
    return result;
}

jayess_value *jayess_std_crypto_secure_compare(jayess_value *left, jayess_value *right) {
    unsigned char *left_bytes = NULL;
    unsigned char *right_bytes = NULL;
    size_t left_length = 0;
    size_t right_length = 0;
    size_t i;
    unsigned char diff = 0;
    size_t max_length;
    if (!jayess_std_crypto_copy_bytes(left, &left_bytes, &left_length) ||
        !jayess_std_crypto_copy_bytes(right, &right_bytes, &right_length)) {
        free(left_bytes);
        free(right_bytes);
        return jayess_value_from_bool(0);
    }
    max_length = left_length > right_length ? left_length : right_length;
    diff = (unsigned char)(left_length ^ right_length);
    for (i = 0; i < max_length; i++) {
        unsigned char left_byte = i < left_length ? left_bytes[i] : 0;
        unsigned char right_byte = i < right_length ? right_bytes[i] : 0;
        diff |= (unsigned char)(left_byte ^ right_byte);
    }
    free(left_bytes);
    free(right_bytes);
    return jayess_value_from_bool(diff == 0);
}

jayess_value *jayess_std_crypto_encrypt(jayess_value *options) {
    jayess_object *object = jayess_value_as_object(options);
    char *algorithm = NULL;
    unsigned char *key = NULL;
    unsigned char *iv = NULL;
    unsigned char *data = NULL;
    unsigned char *aad = NULL;
    size_t key_length = 0;
    size_t iv_length = 0;
    size_t data_length = 0;
    size_t aad_length = 0;
    int expected_key_length;
    jayess_object *result = NULL;
    jayess_value *boxed = NULL;
    if (object == NULL) {
        return jayess_value_undefined();
    }
    algorithm = jayess_compile_option_string(options, "algorithm");
    expected_key_length = jayess_std_crypto_cipher_key_length(algorithm);
    if (expected_key_length == 0 ||
        !jayess_std_crypto_option_bytes(options, "key", &key, &key_length, 1) ||
        !jayess_std_crypto_option_bytes(options, "iv", &iv, &iv_length, 1) ||
        !jayess_std_crypto_option_bytes(options, "data", &data, &data_length, 1) ||
        !jayess_std_crypto_option_bytes(options, "aad", &aad, &aad_length, 0) ||
        (int)key_length != expected_key_length || iv_length == 0) {
        free(algorithm);
        free(key);
        free(iv);
        free(data);
        free(aad);
        return jayess_value_undefined();
    }
#ifdef _WIN32
    {
        BCRYPT_ALG_HANDLE provider = NULL;
        BCRYPT_KEY_HANDLE key_handle = NULL;
        DWORD object_length = 0;
        DWORD bytes_written = 0;
        PUCHAR key_object = NULL;
        unsigned char *ciphertext = NULL;
        ULONG ciphertext_length = 0;
        unsigned char tag[16];
        BCRYPT_AUTHENTICATED_CIPHER_MODE_INFO auth_info;
        if (BCryptOpenAlgorithmProvider(&provider, BCRYPT_AES_ALGORITHM, NULL, 0) < 0 ||
            BCryptSetProperty(provider, BCRYPT_CHAINING_MODE, (PUCHAR)BCRYPT_CHAIN_MODE_GCM, (ULONG)(sizeof(BCRYPT_CHAIN_MODE_GCM)), 0) < 0 ||
            BCryptGetProperty(provider, BCRYPT_OBJECT_LENGTH, (PUCHAR)&object_length, sizeof(object_length), &bytes_written, 0) < 0) {
            if (provider != NULL) {
                BCryptCloseAlgorithmProvider(provider, 0);
            }
            free(algorithm);
            free(key);
            free(iv);
            free(data);
            free(aad);
            return jayess_value_undefined();
        }
        key_object = (PUCHAR)malloc(object_length > 0 ? object_length : 1);
        ciphertext = (unsigned char *)malloc(data_length > 0 ? data_length : 1);
        if (key_object == NULL || ciphertext == NULL ||
            BCryptGenerateSymmetricKey(provider, &key_handle, key_object, object_length, key, (ULONG)key_length, 0) < 0) {
            if (key_handle != NULL) {
                BCryptDestroyKey(key_handle);
            }
            BCryptCloseAlgorithmProvider(provider, 0);
            free(key_object);
            free(ciphertext);
            free(algorithm);
            free(key);
            free(iv);
            free(data);
            free(aad);
            return jayess_value_undefined();
        }
        BCRYPT_INIT_AUTH_MODE_INFO(auth_info);
        auth_info.pbNonce = iv;
        auth_info.cbNonce = (ULONG)iv_length;
        auth_info.pbAuthData = aad;
        auth_info.cbAuthData = (ULONG)aad_length;
        auth_info.pbTag = tag;
        auth_info.cbTag = (ULONG)sizeof(tag);
        if (BCryptEncrypt(key_handle, data, (ULONG)data_length, &auth_info, NULL, 0, ciphertext, (ULONG)data_length, &ciphertext_length, 0) < 0) {
            BCryptDestroyKey(key_handle);
            BCryptCloseAlgorithmProvider(provider, 0);
            free(key_object);
            free(ciphertext);
            free(algorithm);
            free(key);
            free(iv);
            free(data);
            free(aad);
            return jayess_value_undefined();
        }
        result = jayess_object_new();
        if (result != NULL) {
            jayess_object_set_value(result, "algorithm", jayess_value_from_string(algorithm));
            jayess_object_set_value(result, "iv", jayess_std_uint8_array_from_bytes(iv, iv_length));
            jayess_object_set_value(result, "ciphertext", jayess_std_uint8_array_from_bytes(ciphertext, (size_t)ciphertext_length));
            jayess_object_set_value(result, "tag", jayess_std_uint8_array_from_bytes(tag, sizeof(tag)));
            boxed = jayess_value_from_object(result);
        } else {
            boxed = jayess_value_undefined();
        }
        BCryptDestroyKey(key_handle);
        BCryptCloseAlgorithmProvider(provider, 0);
        free(key_object);
        free(ciphertext);
    }
#else
    {
        const EVP_CIPHER *cipher = NULL;
        EVP_CIPHER_CTX *ctx = NULL;
        unsigned char *ciphertext = NULL;
        int out_length = 0;
        int final_length = 0;
        unsigned char tag[16];
        if (jayess_std_crypto_equal_name(algorithm, "aes-128-gcm")) {
            cipher = EVP_aes_128_gcm();
        } else if (jayess_std_crypto_equal_name(algorithm, "aes-192-gcm")) {
            cipher = EVP_aes_192_gcm();
        } else if (jayess_std_crypto_equal_name(algorithm, "aes-256-gcm")) {
            cipher = EVP_aes_256_gcm();
        }
        if (cipher == NULL) {
            free(algorithm);
            free(key);
            free(iv);
            free(data);
            free(aad);
            return jayess_value_undefined();
        }
        ctx = EVP_CIPHER_CTX_new();
        ciphertext = (unsigned char *)malloc(data_length > 0 ? data_length : 1);
        if (ctx == NULL || ciphertext == NULL ||
            EVP_EncryptInit_ex(ctx, cipher, NULL, NULL, NULL) != 1 ||
            EVP_CIPHER_CTX_ctrl(ctx, EVP_CTRL_GCM_SET_IVLEN, (int)iv_length, NULL) != 1 ||
            EVP_EncryptInit_ex(ctx, NULL, NULL, key, iv) != 1) {
            if (ctx != NULL) {
                EVP_CIPHER_CTX_free(ctx);
            }
            free(ciphertext);
            free(algorithm);
            free(key);
            free(iv);
            free(data);
            free(aad);
            return jayess_value_undefined();
        }
        if (aad_length > 0 && EVP_EncryptUpdate(ctx, NULL, &out_length, aad, (int)aad_length) != 1) {
            EVP_CIPHER_CTX_free(ctx);
            free(ciphertext);
            free(algorithm);
            free(key);
            free(iv);
            free(data);
            free(aad);
            return jayess_value_undefined();
        }
        if ((data_length > 0 && EVP_EncryptUpdate(ctx, ciphertext, &out_length, data, (int)data_length) != 1) ||
            EVP_EncryptFinal_ex(ctx, ciphertext + out_length, &final_length) != 1 ||
            EVP_CIPHER_CTX_ctrl(ctx, EVP_CTRL_GCM_GET_TAG, (int)sizeof(tag), tag) != 1) {
            EVP_CIPHER_CTX_free(ctx);
            free(ciphertext);
            free(algorithm);
            free(key);
            free(iv);
            free(data);
            free(aad);
            return jayess_value_undefined();
        }
        result = jayess_object_new();
        if (result != NULL) {
            jayess_object_set_value(result, "algorithm", jayess_value_from_string(algorithm));
            jayess_object_set_value(result, "iv", jayess_std_uint8_array_from_bytes(iv, iv_length));
            jayess_object_set_value(result, "ciphertext", jayess_std_uint8_array_from_bytes(ciphertext, (size_t)(out_length + final_length)));
            jayess_object_set_value(result, "tag", jayess_std_uint8_array_from_bytes(tag, sizeof(tag)));
            boxed = jayess_value_from_object(result);
        } else {
            boxed = jayess_value_undefined();
        }
        EVP_CIPHER_CTX_free(ctx);
        free(ciphertext);
    }
#endif
    free(algorithm);
    free(key);
    free(iv);
    free(data);
    free(aad);
    return boxed != NULL ? boxed : jayess_value_undefined();
}

jayess_value *jayess_std_crypto_decrypt(jayess_value *options) {
    jayess_object *object = jayess_value_as_object(options);
    char *algorithm = NULL;
    unsigned char *key = NULL;
    unsigned char *iv = NULL;
    unsigned char *data = NULL;
    unsigned char *tag = NULL;
    unsigned char *aad = NULL;
    size_t key_length = 0;
    size_t iv_length = 0;
    size_t data_length = 0;
    size_t tag_length = 0;
    size_t aad_length = 0;
    int expected_key_length;
    jayess_value *boxed = NULL;
    if (object == NULL) {
        return jayess_value_undefined();
    }
    algorithm = jayess_compile_option_string(options, "algorithm");
    expected_key_length = jayess_std_crypto_cipher_key_length(algorithm);
    if (expected_key_length == 0 ||
        !jayess_std_crypto_option_bytes(options, "key", &key, &key_length, 1) ||
        !jayess_std_crypto_option_bytes(options, "iv", &iv, &iv_length, 1) ||
        !jayess_std_crypto_option_bytes(options, "data", &data, &data_length, 1) ||
        !jayess_std_crypto_option_bytes(options, "tag", &tag, &tag_length, 1) ||
        !jayess_std_crypto_option_bytes(options, "aad", &aad, &aad_length, 0) ||
        (int)key_length != expected_key_length || iv_length == 0 || tag_length != 16) {
        free(algorithm);
        free(key);
        free(iv);
        free(data);
        free(tag);
        free(aad);
        return jayess_value_undefined();
    }
#ifdef _WIN32
    {
        BCRYPT_ALG_HANDLE provider = NULL;
        BCRYPT_KEY_HANDLE key_handle = NULL;
        DWORD object_length = 0;
        DWORD bytes_written = 0;
        PUCHAR key_object = NULL;
        unsigned char *plaintext = NULL;
        ULONG plaintext_length = 0;
        BCRYPT_AUTHENTICATED_CIPHER_MODE_INFO auth_info;
        if (BCryptOpenAlgorithmProvider(&provider, BCRYPT_AES_ALGORITHM, NULL, 0) < 0 ||
            BCryptSetProperty(provider, BCRYPT_CHAINING_MODE, (PUCHAR)BCRYPT_CHAIN_MODE_GCM, (ULONG)(sizeof(BCRYPT_CHAIN_MODE_GCM)), 0) < 0 ||
            BCryptGetProperty(provider, BCRYPT_OBJECT_LENGTH, (PUCHAR)&object_length, sizeof(object_length), &bytes_written, 0) < 0) {
            if (provider != NULL) {
                BCryptCloseAlgorithmProvider(provider, 0);
            }
            free(algorithm);
            free(key);
            free(iv);
            free(data);
            free(tag);
            free(aad);
            return jayess_value_undefined();
        }
        key_object = (PUCHAR)malloc(object_length > 0 ? object_length : 1);
        plaintext = (unsigned char *)malloc(data_length > 0 ? data_length : 1);
        if (key_object == NULL || plaintext == NULL ||
            BCryptGenerateSymmetricKey(provider, &key_handle, key_object, object_length, key, (ULONG)key_length, 0) < 0) {
            if (key_handle != NULL) {
                BCryptDestroyKey(key_handle);
            }
            BCryptCloseAlgorithmProvider(provider, 0);
            free(key_object);
            free(plaintext);
            free(algorithm);
            free(key);
            free(iv);
            free(data);
            free(tag);
            free(aad);
            return jayess_value_undefined();
        }
        BCRYPT_INIT_AUTH_MODE_INFO(auth_info);
        auth_info.pbNonce = iv;
        auth_info.cbNonce = (ULONG)iv_length;
        auth_info.pbAuthData = aad;
        auth_info.cbAuthData = (ULONG)aad_length;
        auth_info.pbTag = tag;
        auth_info.cbTag = (ULONG)tag_length;
        if (BCryptDecrypt(key_handle, data, (ULONG)data_length, &auth_info, NULL, 0, plaintext, (ULONG)data_length, &plaintext_length, 0) < 0) {
            BCryptDestroyKey(key_handle);
            BCryptCloseAlgorithmProvider(provider, 0);
            free(key_object);
            free(plaintext);
            free(algorithm);
            free(key);
            free(iv);
            free(data);
            free(tag);
            free(aad);
            return jayess_value_undefined();
        }
        boxed = jayess_std_uint8_array_from_bytes(plaintext, (size_t)plaintext_length);
        BCryptDestroyKey(key_handle);
        BCryptCloseAlgorithmProvider(provider, 0);
        free(key_object);
        free(plaintext);
    }
#else
    {
        const EVP_CIPHER *cipher = NULL;
        EVP_CIPHER_CTX *ctx = NULL;
        unsigned char *plaintext = NULL;
        int out_length = 0;
        int final_length = 0;
        int ok = 0;
        if (jayess_std_crypto_equal_name(algorithm, "aes-128-gcm")) {
            cipher = EVP_aes_128_gcm();
        } else if (jayess_std_crypto_equal_name(algorithm, "aes-192-gcm")) {
            cipher = EVP_aes_192_gcm();
        } else if (jayess_std_crypto_equal_name(algorithm, "aes-256-gcm")) {
            cipher = EVP_aes_256_gcm();
        }
        if (cipher == NULL) {
            free(algorithm);
            free(key);
            free(iv);
            free(data);
            free(tag);
            free(aad);
            return jayess_value_undefined();
        }
        ctx = EVP_CIPHER_CTX_new();
        plaintext = (unsigned char *)malloc(data_length > 0 ? data_length : 1);
        if (ctx == NULL || plaintext == NULL ||
            EVP_DecryptInit_ex(ctx, cipher, NULL, NULL, NULL) != 1 ||
            EVP_CIPHER_CTX_ctrl(ctx, EVP_CTRL_GCM_SET_IVLEN, (int)iv_length, NULL) != 1 ||
            EVP_DecryptInit_ex(ctx, NULL, NULL, key, iv) != 1) {
            if (ctx != NULL) {
                EVP_CIPHER_CTX_free(ctx);
            }
            free(plaintext);
            free(algorithm);
            free(key);
            free(iv);
            free(data);
            free(tag);
            free(aad);
            return jayess_value_undefined();
        }
        if (aad_length > 0 && EVP_DecryptUpdate(ctx, NULL, &out_length, aad, (int)aad_length) != 1) {
            EVP_CIPHER_CTX_free(ctx);
            free(plaintext);
            free(algorithm);
            free(key);
            free(iv);
            free(data);
            free(tag);
            free(aad);
            return jayess_value_undefined();
        }
        if ((data_length > 0 && EVP_DecryptUpdate(ctx, plaintext, &out_length, data, (int)data_length) != 1) ||
            EVP_CIPHER_CTX_ctrl(ctx, EVP_CTRL_GCM_SET_TAG, (int)tag_length, tag) != 1) {
            EVP_CIPHER_CTX_free(ctx);
            free(plaintext);
            free(algorithm);
            free(key);
            free(iv);
            free(data);
            free(tag);
            free(aad);
            return jayess_value_undefined();
        }
        ok = EVP_DecryptFinal_ex(ctx, plaintext + out_length, &final_length);
        if (ok != 1) {
            EVP_CIPHER_CTX_free(ctx);
            free(plaintext);
            free(algorithm);
            free(key);
            free(iv);
            free(data);
            free(tag);
            free(aad);
            return jayess_value_undefined();
        }
        boxed = jayess_std_uint8_array_from_bytes(plaintext, (size_t)(out_length + final_length));
        EVP_CIPHER_CTX_free(ctx);
        free(plaintext);
    }
#endif
    free(algorithm);
    free(key);
    free(iv);
    free(data);
    free(tag);
    free(aad);
    return boxed != NULL ? boxed : jayess_value_undefined();
}

jayess_value *jayess_std_crypto_generate_key_pair(jayess_value *options) {
    jayess_object *object = jayess_value_as_object(options);
    char *type = NULL;
    int modulus_length = 2048;
    jayess_value *public_key;
    jayess_value *private_key;
    jayess_crypto_key_state *public_state;
    jayess_crypto_key_state *private_state;
    jayess_object *result;
    if (object == NULL) {
        return jayess_value_undefined();
    }
    type = jayess_compile_option_string(options, "type");
    if (!jayess_std_crypto_equal_name(type, "rsa")) {
        free(type);
        return jayess_value_undefined();
    }
    if (jayess_object_get(object, "modulusLength") != NULL) {
        modulus_length = (int)jayess_value_to_number(jayess_object_get(object, "modulusLength"));
    }
    if (modulus_length < 1024) {
        modulus_length = 1024;
    }
    public_key = jayess_std_crypto_key_value("rsa", 0);
    private_key = jayess_std_crypto_key_value("rsa", 1);
    public_state = jayess_std_crypto_key_state_from_value(public_key);
    private_state = jayess_std_crypto_key_state_from_value(private_key);
    if (public_state == NULL || private_state == NULL) {
        free(type);
        return jayess_value_undefined();
    }
#ifdef _WIN32
    {
        BCRYPT_ALG_HANDLE provider = NULL;
        BCRYPT_KEY_HANDLE private_handle = NULL;
        BCRYPT_KEY_HANDLE public_handle = NULL;
        DWORD blob_length = 0;
        unsigned char *blob = NULL;
        if (BCryptOpenAlgorithmProvider(&provider, BCRYPT_RSA_ALGORITHM, NULL, 0) < 0 ||
            BCryptGenerateKeyPair(provider, &private_handle, (ULONG)modulus_length, 0) < 0 ||
            BCryptFinalizeKeyPair(private_handle, 0) < 0 ||
            BCryptExportKey(private_handle, NULL, BCRYPT_RSAPUBLIC_BLOB, NULL, 0, &blob_length, 0) < 0) {
            if (private_handle != NULL) {
                BCryptDestroyKey(private_handle);
            }
            if (provider != NULL) {
                BCryptCloseAlgorithmProvider(provider, 0);
            }
            free(type);
            return jayess_value_undefined();
        }
        blob = (unsigned char *)malloc(blob_length > 0 ? blob_length : 1);
        if (blob == NULL ||
            BCryptExportKey(private_handle, NULL, BCRYPT_RSAPUBLIC_BLOB, blob, blob_length, &blob_length, 0) < 0 ||
            BCryptImportKeyPair(provider, NULL, BCRYPT_RSAPUBLIC_BLOB, &public_handle, blob, blob_length, 0) < 0) {
            BCryptDestroyKey(private_handle);
            BCryptCloseAlgorithmProvider(provider, 0);
            free(blob);
            free(type);
            return jayess_value_undefined();
        }
        private_state->handle = private_handle;
        public_state->handle = public_handle;
        BCryptCloseAlgorithmProvider(provider, 0);
        free(blob);
    }
#else
    {
        EVP_PKEY_CTX *ctx = EVP_PKEY_CTX_new_id(EVP_PKEY_RSA, NULL);
        EVP_PKEY *pkey = NULL;
        if (ctx == NULL ||
            EVP_PKEY_keygen_init(ctx) <= 0 ||
            EVP_PKEY_CTX_set_rsa_keygen_bits(ctx, modulus_length) <= 0 ||
            EVP_PKEY_keygen(ctx, &pkey) <= 0 ||
            pkey == NULL) {
            if (ctx != NULL) {
                EVP_PKEY_CTX_free(ctx);
            }
            free(type);
            return jayess_value_undefined();
        }
        EVP_PKEY_up_ref(pkey);
        private_state->pkey = pkey;
        public_state->pkey = pkey;
        EVP_PKEY_CTX_free(ctx);
    }
#endif
    result = jayess_object_new();
    free(type);
    if (result == NULL) {
        return jayess_value_undefined();
    }
    jayess_object_set_value(result, "publicKey", public_key);
    jayess_object_set_value(result, "privateKey", private_key);
    return jayess_value_from_object(result);
}

jayess_value *jayess_std_crypto_public_encrypt(jayess_value *options) {
    jayess_object *object = jayess_value_as_object(options);
    char *algorithm = NULL;
    unsigned char *data = NULL;
    size_t data_length = 0;
    jayess_crypto_key_state *key_state = NULL;
    jayess_value *key_value;
    jayess_value *boxed = NULL;
    if (object == NULL) {
        return jayess_value_undefined();
    }
    algorithm = jayess_compile_option_string(options, "algorithm");
    key_value = jayess_object_get(object, "key");
    key_state = jayess_std_crypto_key_state_from_value(key_value);
    if (!jayess_std_crypto_equal_name(algorithm, "rsa-oaep-sha256") || key_state == NULL ||
        !jayess_std_crypto_option_bytes(options, "data", &data, &data_length, 1)) {
        free(algorithm);
        free(data);
        return jayess_value_undefined();
    }
#ifdef _WIN32
    {
        BCRYPT_OAEP_PADDING_INFO padding = { BCRYPT_SHA256_ALGORITHM, NULL, 0 };
        ULONG out_length = 0;
        unsigned char *ciphertext = NULL;
        if (BCryptEncrypt(key_state->handle, data, (ULONG)data_length, &padding, NULL, 0, NULL, 0, &out_length, BCRYPT_PAD_OAEP) < 0) {
            free(algorithm);
            free(data);
            return jayess_value_undefined();
        }
        ciphertext = (unsigned char *)malloc(out_length > 0 ? out_length : 1);
        if (ciphertext == NULL ||
            BCryptEncrypt(key_state->handle, data, (ULONG)data_length, &padding, NULL, 0, ciphertext, out_length, &out_length, BCRYPT_PAD_OAEP) < 0) {
            free(ciphertext);
            free(algorithm);
            free(data);
            return jayess_value_undefined();
        }
        boxed = jayess_std_uint8_array_from_bytes(ciphertext, (size_t)out_length);
        free(ciphertext);
    }
#else
    {
        EVP_PKEY_CTX *ctx = EVP_PKEY_CTX_new(key_state->pkey, NULL);
        size_t out_length = 0;
        unsigned char *ciphertext = NULL;
        if (ctx == NULL ||
            EVP_PKEY_encrypt_init(ctx) <= 0 ||
            EVP_PKEY_CTX_set_rsa_padding(ctx, RSA_PKCS1_OAEP_PADDING) <= 0 ||
            EVP_PKEY_CTX_set_rsa_oaep_md(ctx, EVP_sha256()) <= 0 ||
            EVP_PKEY_CTX_set_rsa_mgf1_md(ctx, EVP_sha256()) <= 0 ||
            EVP_PKEY_encrypt(ctx, NULL, &out_length, data, data_length) <= 0) {
            if (ctx != NULL) {
                EVP_PKEY_CTX_free(ctx);
            }
            free(algorithm);
            free(data);
            return jayess_value_undefined();
        }
        ciphertext = (unsigned char *)malloc(out_length > 0 ? out_length : 1);
        if (ciphertext == NULL ||
            EVP_PKEY_encrypt(ctx, ciphertext, &out_length, data, data_length) <= 0) {
            EVP_PKEY_CTX_free(ctx);
            free(ciphertext);
            free(algorithm);
            free(data);
            return jayess_value_undefined();
        }
        boxed = jayess_std_uint8_array_from_bytes(ciphertext, out_length);
        EVP_PKEY_CTX_free(ctx);
        free(ciphertext);
    }
#endif
    free(algorithm);
    free(data);
    return boxed != NULL ? boxed : jayess_value_undefined();
}

jayess_value *jayess_std_crypto_private_decrypt(jayess_value *options) {
    jayess_object *object = jayess_value_as_object(options);
    char *algorithm = NULL;
    unsigned char *data = NULL;
    size_t data_length = 0;
    jayess_crypto_key_state *key_state = NULL;
    jayess_value *key_value;
    jayess_value *boxed = NULL;
    if (object == NULL) {
        return jayess_value_undefined();
    }
    algorithm = jayess_compile_option_string(options, "algorithm");
    key_value = jayess_object_get(object, "key");
    key_state = jayess_std_crypto_key_state_from_value(key_value);
    if (!jayess_std_crypto_equal_name(algorithm, "rsa-oaep-sha256") || key_state == NULL || !key_state->is_private ||
        !jayess_std_crypto_option_bytes(options, "data", &data, &data_length, 1)) {
        free(algorithm);
        free(data);
        return jayess_value_undefined();
    }
#ifdef _WIN32
    {
        BCRYPT_OAEP_PADDING_INFO padding = { BCRYPT_SHA256_ALGORITHM, NULL, 0 };
        ULONG out_length = 0;
        unsigned char *plaintext = NULL;
        if (BCryptDecrypt(key_state->handle, data, (ULONG)data_length, &padding, NULL, 0, NULL, 0, &out_length, BCRYPT_PAD_OAEP) < 0) {
            free(algorithm);
            free(data);
            return jayess_value_undefined();
        }
        plaintext = (unsigned char *)malloc(out_length > 0 ? out_length : 1);
        if (plaintext == NULL ||
            BCryptDecrypt(key_state->handle, data, (ULONG)data_length, &padding, NULL, 0, plaintext, out_length, &out_length, BCRYPT_PAD_OAEP) < 0) {
            free(plaintext);
            free(algorithm);
            free(data);
            return jayess_value_undefined();
        }
        boxed = jayess_std_uint8_array_from_bytes(plaintext, (size_t)out_length);
        free(plaintext);
    }
#else
    {
        EVP_PKEY_CTX *ctx = EVP_PKEY_CTX_new(key_state->pkey, NULL);
        size_t out_length = 0;
        unsigned char *plaintext = NULL;
        if (ctx == NULL ||
            EVP_PKEY_decrypt_init(ctx) <= 0 ||
            EVP_PKEY_CTX_set_rsa_padding(ctx, RSA_PKCS1_OAEP_PADDING) <= 0 ||
            EVP_PKEY_CTX_set_rsa_oaep_md(ctx, EVP_sha256()) <= 0 ||
            EVP_PKEY_CTX_set_rsa_mgf1_md(ctx, EVP_sha256()) <= 0 ||
            EVP_PKEY_decrypt(ctx, NULL, &out_length, data, data_length) <= 0) {
            if (ctx != NULL) {
                EVP_PKEY_CTX_free(ctx);
            }
            free(algorithm);
            free(data);
            return jayess_value_undefined();
        }
        plaintext = (unsigned char *)malloc(out_length > 0 ? out_length : 1);
        if (plaintext == NULL ||
            EVP_PKEY_decrypt(ctx, plaintext, &out_length, data, data_length) <= 0) {
            EVP_PKEY_CTX_free(ctx);
            free(plaintext);
            free(algorithm);
            free(data);
            return jayess_value_undefined();
        }
        boxed = jayess_std_uint8_array_from_bytes(plaintext, out_length);
        EVP_PKEY_CTX_free(ctx);
        free(plaintext);
    }
#endif
    free(algorithm);
    free(data);
    return boxed != NULL ? boxed : jayess_value_undefined();
}

jayess_value *jayess_std_crypto_sign(jayess_value *options) {
    jayess_object *object = jayess_value_as_object(options);
    char *algorithm = NULL;
    unsigned char *data = NULL;
    size_t data_length = 0;
    jayess_crypto_key_state *key_state = NULL;
    jayess_value *key_value;
    jayess_value *boxed = NULL;
    if (object == NULL) {
        return jayess_value_undefined();
    }
    algorithm = jayess_compile_option_string(options, "algorithm");
    key_value = jayess_object_get(object, "key");
    key_state = jayess_std_crypto_key_state_from_value(key_value);
    if (!jayess_std_crypto_equal_name(algorithm, "rsa-pss-sha256") || key_state == NULL || !key_state->is_private ||
        !jayess_std_crypto_option_bytes(options, "data", &data, &data_length, 1)) {
        free(algorithm);
        free(data);
        return jayess_value_undefined();
    }
#ifdef _WIN32
    {
        BCRYPT_PSS_PADDING_INFO padding = { BCRYPT_SHA256_ALGORITHM, 32 };
        unsigned char digest[32];
        DWORD digest_length = sizeof(digest);
        ULONG signature_length = 0;
        unsigned char *signature = NULL;
        if (!jayess_std_crypto_sha256_bytes(data, data_length, digest, &digest_length) ||
            BCryptSignHash(key_state->handle, &padding, digest, digest_length, NULL, 0, &signature_length, BCRYPT_PAD_PSS) < 0) {
            free(algorithm);
            free(data);
            return jayess_value_undefined();
        }
        signature = (unsigned char *)malloc(signature_length > 0 ? signature_length : 1);
        if (signature == NULL ||
            BCryptSignHash(key_state->handle, &padding, digest, digest_length, signature, signature_length, &signature_length, BCRYPT_PAD_PSS) < 0) {
            free(signature);
            free(algorithm);
            free(data);
            return jayess_value_undefined();
        }
        boxed = jayess_std_uint8_array_from_bytes(signature, (size_t)signature_length);
        free(signature);
    }
#else
    {
        EVP_MD_CTX *ctx = EVP_MD_CTX_new();
        EVP_PKEY_CTX *pkey_ctx = NULL;
        size_t signature_length = 0;
        unsigned char *signature = NULL;
        if (ctx == NULL ||
            EVP_DigestSignInit(ctx, &pkey_ctx, EVP_sha256(), NULL, key_state->pkey) <= 0 ||
            EVP_PKEY_CTX_set_rsa_padding(pkey_ctx, RSA_PKCS1_PSS_PADDING) <= 0 ||
            EVP_PKEY_CTX_set_rsa_mgf1_md(pkey_ctx, EVP_sha256()) <= 0 ||
            EVP_DigestSignUpdate(ctx, data, data_length) <= 0 ||
            EVP_DigestSignFinal(ctx, NULL, &signature_length) <= 0) {
            if (ctx != NULL) {
                EVP_MD_CTX_free(ctx);
            }
            free(algorithm);
            free(data);
            return jayess_value_undefined();
        }
        signature = (unsigned char *)malloc(signature_length > 0 ? signature_length : 1);
        if (signature == NULL ||
            EVP_DigestSignFinal(ctx, signature, &signature_length) <= 0) {
            EVP_MD_CTX_free(ctx);
            free(signature);
            free(algorithm);
            free(data);
            return jayess_value_undefined();
        }
        boxed = jayess_std_uint8_array_from_bytes(signature, signature_length);
        EVP_MD_CTX_free(ctx);
        free(signature);
    }
#endif
    free(algorithm);
    free(data);
    return boxed != NULL ? boxed : jayess_value_undefined();
}

jayess_value *jayess_std_crypto_verify(jayess_value *options) {
    jayess_object *object = jayess_value_as_object(options);
    char *algorithm = NULL;
    unsigned char *data = NULL;
    unsigned char *signature = NULL;
    size_t data_length = 0;
    size_t signature_length = 0;
    jayess_crypto_key_state *key_state = NULL;
    jayess_value *key_value;
    int ok = 0;
    if (object == NULL) {
        return jayess_value_undefined();
    }
    algorithm = jayess_compile_option_string(options, "algorithm");
    key_value = jayess_object_get(object, "key");
    key_state = jayess_std_crypto_key_state_from_value(key_value);
    if (!jayess_std_crypto_equal_name(algorithm, "rsa-pss-sha256") || key_state == NULL ||
        !jayess_std_crypto_option_bytes(options, "data", &data, &data_length, 1) ||
        !jayess_std_crypto_option_bytes(options, "signature", &signature, &signature_length, 1)) {
        free(algorithm);
        free(data);
        free(signature);
        return jayess_value_undefined();
    }
#ifdef _WIN32
    {
        BCRYPT_PSS_PADDING_INFO padding = { BCRYPT_SHA256_ALGORITHM, 32 };
        unsigned char digest[32];
        DWORD digest_length = sizeof(digest);
        if (jayess_std_crypto_sha256_bytes(data, data_length, digest, &digest_length) &&
            BCryptVerifySignature(key_state->handle, &padding, digest, digest_length, signature, (ULONG)signature_length, BCRYPT_PAD_PSS) == 0) {
            ok = 1;
        }
    }
#else
    {
        EVP_MD_CTX *ctx = EVP_MD_CTX_new();
        EVP_PKEY_CTX *pkey_ctx = NULL;
        if (ctx != NULL &&
            EVP_DigestVerifyInit(ctx, &pkey_ctx, EVP_sha256(), NULL, key_state->pkey) > 0 &&
            EVP_PKEY_CTX_set_rsa_padding(pkey_ctx, RSA_PKCS1_PSS_PADDING) > 0 &&
            EVP_PKEY_CTX_set_rsa_mgf1_md(pkey_ctx, EVP_sha256()) > 0 &&
            EVP_DigestVerifyUpdate(ctx, data, data_length) > 0 &&
            EVP_DigestVerifyFinal(ctx, signature, signature_length) == 1) {
            ok = 1;
        }
        if (ctx != NULL) {
            EVP_MD_CTX_free(ctx);
        }
    }
#endif
    free(algorithm);
    free(data);
    free(signature);
    return jayess_value_from_bool(ok);
}

jayess_value *jayess_std_path_join(jayess_value *parts) {
    char sep_char = jayess_path_separator_char();
    char sep_text[2] = {0, 0};
    size_t total = 1;
    char *out;
    int i;
    if (parts == NULL || parts->kind != JAYESS_VALUE_ARRAY || parts->as.array_value == NULL) {
        return jayess_value_from_string("");
    }
    for (i = 0; i < parts->as.array_value->count; i++) {
        char *piece = jayess_value_stringify(jayess_array_get(parts->as.array_value, i));
        if (piece != NULL) {
            char piece_sep = jayess_path_preferred_separator_char(piece);
            if (piece_sep == '\\') {
                sep_char = '\\';
            }
        }
        free(piece);
    }
    sep_text[0] = sep_char;
    for (i = 0; i < parts->as.array_value->count; i++) {
        char *piece = jayess_value_stringify(jayess_array_get(parts->as.array_value, i));
        total += strlen(piece != NULL ? piece : "");
        if (i > 0) {
            total += 1;
        }
        free(piece);
    }
    out = (char *)malloc(total);
    if (out == NULL) {
        return jayess_value_from_string("");
    }
    out[0] = '\0';
    for (i = 0; i < parts->as.array_value->count; i++) {
        char *piece = jayess_value_stringify(jayess_array_get(parts->as.array_value, i));
        if (i > 0) {
            strcat(out, sep_text);
        }
        strcat(out, piece != NULL ? piece : "");
        free(piece);
    }
    parts = jayess_value_from_string(out);
    free(out);
    return parts;
}

jayess_value *jayess_std_path_normalize(jayess_value *path) {
    char *path_text = jayess_value_stringify(path);
    int absolute = jayess_path_is_absolute(path_text);
    int root_length = jayess_path_root_length(path_text);
    char sep_char = jayess_path_preferred_separator_char(path_text);
    jayess_array *segments = jayess_array_new();
    const char *cursor = path_text != NULL ? path_text + root_length : "";
    char *root_text = NULL;
    if (root_length > 0) {
        root_text = (char *)malloc((size_t)root_length + 1);
        if (root_text != NULL) {
            memcpy(root_text, path_text, (size_t)root_length);
            root_text[root_length] = '\0';
            if (root_length >= 2 && isalpha((unsigned char)root_text[0]) && root_text[1] == ':') {
                if (root_length == 2) {
                    root_text = (char *)realloc(root_text, 4);
                    if (root_text != NULL) {
                        root_text[2] = sep_char;
                        root_text[3] = '\0';
                    }
                } else {
                    root_text[root_length - 1] = sep_char;
                }
            }
        }
    }
    while (*cursor != '\0') {
        const char *start = cursor;
        while (*cursor != '\0' && !jayess_path_is_separator(*cursor)) {
            cursor++;
        }
        if (cursor > start) {
            size_t length = (size_t)(cursor - start);
            char *segment = (char *)malloc(length + 1);
            jayess_value *value;
            if (segment == NULL) {
                free(path_text);
                return jayess_value_from_string(path_text != NULL ? path_text : "");
            }
            memcpy(segment, start, length);
            segment[length] = '\0';
            if (strcmp(segment, ".") == 0) {
                free(segment);
            } else if (strcmp(segment, "..") == 0) {
                if (segments->count > 0) {
                    jayess_array_pop_value(segments);
                }
                free(segment);
            } else {
                value = jayess_value_from_string(segment);
                jayess_array_push_value(segments, value);
                free(segment);
            }
        }
        while (*cursor != '\0' && jayess_path_is_separator(*cursor)) {
            cursor++;
        }
    }
    {
        jayess_value *joined = jayess_std_path_join(jayess_value_from_array(segments));
        char *joined_text = jayess_value_stringify(joined);
        jayess_value *result;
        if (joined_text == NULL) {
            free(path_text);
            free(root_text);
            if (absolute) {
                char fallback[2] = {sep_char, '\0'};
                return jayess_value_from_string(fallback);
            }
            return jayess_value_from_string(".");
        }
        if (absolute) {
            char *prefixed = jayess_path_join_segments_with_root(root_text != NULL ? root_text : "", segments, sep_char);
            if (prefixed != NULL) {
                free(joined_text);
                joined_text = prefixed;
            }
        }
        if (!absolute && joined_text[0] == '\0') {
            free(joined_text);
            joined_text = jayess_strdup(".");
        }
        result = jayess_value_from_string(joined_text);
        free(joined_text);
        free(path_text);
        free(root_text);
        return result;
    }
}

jayess_value *jayess_std_path_resolve(jayess_value *parts) {
    jayess_array *values = jayess_array_new();
    int i;
    int start = 0;
    if (parts == NULL || parts->kind != JAYESS_VALUE_ARRAY || parts->as.array_value == NULL || parts->as.array_value->count == 0) {
        return jayess_std_process_cwd();
    }
    for (i = parts->as.array_value->count - 1; i >= 0; i--) {
        jayess_value *part = jayess_array_get(parts->as.array_value, i);
        char *text = jayess_value_stringify(part);
        if (text != NULL && text[0] != '\0') {
            if (jayess_path_is_absolute(text)) {
                start = i;
                free(text);
                break;
            }
        }
        free(text);
    }
    if (i < 0) {
        jayess_array_push_value(values, jayess_std_process_cwd());
        start = 0;
    }
    for (i = start; i < parts->as.array_value->count; i++) {
        char *text = jayess_value_stringify(jayess_array_get(parts->as.array_value, i));
        if (text != NULL && text[0] != '\0') {
            jayess_array_push_value(values, jayess_value_from_string(text));
        }
        free(text);
    }
    return jayess_std_path_normalize(jayess_std_path_join(jayess_value_from_array(values)));
}

jayess_value *jayess_std_path_relative(jayess_value *from, jayess_value *to) {
    jayess_array *from_parts = jayess_array_new();
    jayess_array *to_parts = jayess_array_new();
    jayess_value *from_resolved;
    jayess_value *to_resolved;
    char *from_text;
    char *to_text;
    jayess_array *from_segments;
    jayess_array *to_segments;
    jayess_array *relative_segments = jayess_array_new();
    int common = 0;
    int i;
    char *joined;
    if (relative_segments == NULL) {
        return jayess_value_from_string(".");
    }
    jayess_array_push_value(from_parts, from);
    jayess_array_push_value(to_parts, to);
    from_resolved = jayess_std_path_resolve(jayess_value_from_array(from_parts));
    to_resolved = jayess_std_path_resolve(jayess_value_from_array(to_parts));
    from_text = jayess_value_stringify(from_resolved);
    to_text = jayess_value_stringify(to_resolved);
    if (from_text == NULL || to_text == NULL) {
        free(from_text);
        free(to_text);
        return jayess_value_from_string(".");
    }
    from_segments = jayess_path_split_segments(from_text);
    to_segments = jayess_path_split_segments(to_text);
    if (jayess_path_root_length(from_text) != jayess_path_root_length(to_text)) {
        free(from_text);
        free(to_text);
        return to_resolved;
    }
#ifdef _WIN32
    if (_strnicmp(from_text, to_text, (size_t)jayess_path_root_length(from_text)) != 0) {
#else
    if (strncmp(from_text, to_text, (size_t)jayess_path_root_length(from_text)) != 0) {
#endif
        free(from_text);
        free(to_text);
        return to_resolved;
    }
    while (common < from_segments->count && common < to_segments->count) {
        const char *left = jayess_value_as_string(jayess_array_get(from_segments, common));
        const char *right = jayess_value_as_string(jayess_array_get(to_segments, common));
        if (strcmp(left, right) != 0) {
            break;
        }
        common++;
    }
    for (i = common; i < from_segments->count; i++) {
        jayess_array_push_value(relative_segments, jayess_value_from_string(".."));
    }
    for (i = common; i < to_segments->count; i++) {
        jayess_array_push_value(relative_segments, jayess_array_get(to_segments, i));
    }
    {
        char sep_char = jayess_path_preferred_separator_char(to_text);
        joined = jayess_path_join_segments_with_root("", relative_segments, sep_char);
    }
    free(from_text);
    free(to_text);
    if (joined == NULL) {
        return jayess_value_from_string(".");
    }
    from_resolved = jayess_value_from_string(joined);
    free(joined);
    return from_resolved;
}

jayess_value *jayess_std_path_parse(jayess_value *path) {
    char *path_text = jayess_value_stringify(path);
    int root_length;
    const char *last_sep;
    const char *base;
    jayess_object *parsed = jayess_object_new();
    jayess_value *result;
    char *dir_text;
    char *base_text;
    char *ext_text;
    char *name_text;
    if (parsed == NULL) {
        free(path_text);
        return jayess_value_undefined();
    }
    if (path_text == NULL) {
        path_text = jayess_strdup("");
    }
    root_length = jayess_path_root_length(path_text);
    last_sep = jayess_path_last_separator(path_text);
    base = last_sep != NULL ? last_sep + 1 : path_text;
    dir_text = jayess_value_stringify(jayess_std_path_dirname(jayess_value_from_string(path_text)));
    base_text = jayess_strdup(base);
    ext_text = jayess_value_stringify(jayess_std_path_extname(jayess_value_from_string(path_text)));
    if (ext_text != NULL && ext_text[0] != '\0' && strlen(base_text) >= strlen(ext_text)) {
        size_t name_len = strlen(base_text) - strlen(ext_text);
        name_text = (char *)malloc(name_len + 1);
        if (name_text != NULL) {
            memcpy(name_text, base_text, name_len);
            name_text[name_len] = '\0';
        }
    } else {
        name_text = jayess_strdup(base_text != NULL ? base_text : "");
    }
    if (root_length > 0) {
        char *root_text = (char *)malloc((size_t)root_length + 1);
        if (root_text != NULL) {
            memcpy(root_text, path_text, (size_t)root_length);
            root_text[root_length] = '\0';
            jayess_object_set_value(parsed, "root", jayess_value_from_string(root_text));
            free(root_text);
        } else {
            jayess_object_set_value(parsed, "root", jayess_value_from_string(""));
        }
    } else {
        jayess_object_set_value(parsed, "root", jayess_value_from_string(""));
    }
    jayess_object_set_value(parsed, "dir", jayess_value_from_string(dir_text != NULL ? dir_text : "."));
    jayess_object_set_value(parsed, "base", jayess_value_from_string(base_text != NULL ? base_text : ""));
    jayess_object_set_value(parsed, "ext", jayess_value_from_string(ext_text != NULL ? ext_text : ""));
    jayess_object_set_value(parsed, "name", jayess_value_from_string(name_text != NULL ? name_text : ""));
    free(path_text);
    free(dir_text);
    free(base_text);
    free(ext_text);
    free(name_text);
    result = jayess_value_from_object(parsed);
    return result;
}

jayess_value *jayess_std_path_is_absolute(jayess_value *path) {
    char *path_text = jayess_value_stringify(path);
    int absolute = jayess_path_is_absolute(path_text);
    free(path_text);
    return jayess_value_from_bool(absolute);
}

jayess_value *jayess_std_path_format(jayess_value *parts) {
    jayess_object *object = jayess_value_as_object(parts);
    jayess_value *dirValue;
    jayess_value *rootValue;
    jayess_value *baseValue;
    jayess_value *nameValue;
    jayess_value *extValue;
    char *dirText;
    char *rootText;
    char *baseText;
    char *nameText;
    char *extText;
    char *out;
    size_t total;
    char sep = jayess_path_separator_char();
    if (object == NULL) {
        return jayess_value_from_string("");
    }
    dirValue = jayess_object_get(object, "dir");
    rootValue = jayess_object_get(object, "root");
    baseValue = jayess_object_get(object, "base");
    nameValue = jayess_object_get(object, "name");
    extValue = jayess_object_get(object, "ext");
    dirText = jayess_value_stringify(dirValue);
    rootText = jayess_value_stringify(rootValue);
    baseText = jayess_value_stringify(baseValue);
    nameText = jayess_value_stringify(nameValue);
    extText = jayess_value_stringify(extValue);
    if (dirText != NULL && dirText[0] != '\0') {
        sep = jayess_path_preferred_separator_char(dirText);
    } else if (rootText != NULL && rootText[0] != '\0') {
        sep = jayess_path_preferred_separator_char(rootText);
    }
    if ((baseText == NULL || baseText[0] == '\0') && nameText != NULL) {
        size_t nameLen = strlen(nameText);
        size_t extLen = extText != NULL ? strlen(extText) : 0;
        baseText = (char *)realloc(baseText, nameLen + extLen + 1);
        if (baseText != NULL) {
            strcpy(baseText, nameText);
            if (extText != NULL) {
                strcat(baseText, extText);
            }
        }
    }
    total = strlen(dirText != NULL ? dirText : "") + strlen(rootText != NULL ? rootText : "") + strlen(baseText != NULL ? baseText : "") + 2;
    out = (char *)malloc(total);
    if (out == NULL) {
        free(dirText); free(rootText); free(baseText); free(nameText); free(extText);
        return jayess_value_from_string("");
    }
    out[0] = '\0';
    if (dirText != NULL && dirText[0] != '\0') {
        strcpy(out, dirText);
        if (!jayess_path_is_separator(out[strlen(out)-1]) && baseText != NULL && baseText[0] != '\0') {
            size_t len = strlen(out);
            out[len] = sep;
            out[len+1] = '\0';
        }
    } else if (rootText != NULL && rootText[0] != '\0') {
        strcpy(out, rootText);
    }
    if (baseText != NULL) {
        strcat(out, baseText);
    }
    parts = jayess_value_from_string(out);
    free(out);
    free(dirText); free(rootText); free(baseText); free(nameText); free(extText);
    return parts;
}

jayess_value *jayess_std_path_sep(void) {
    return jayess_value_from_string(jayess_path_separator_string());
}

jayess_value *jayess_std_path_delimiter(void) {
    return jayess_value_from_string(jayess_path_delimiter_string());
}

jayess_value *jayess_std_path_basename(jayess_value *path) {
    char *path_text = jayess_value_stringify(path);
    const char *start;
    jayess_value *result;
    if (path_text == NULL) {
        return jayess_value_from_string("");
    }
    start = jayess_path_last_separator(path_text);
    if (start == NULL) {
        result = jayess_value_from_string(path_text);
    } else {
        result = jayess_value_from_string(start + 1);
    }
    free(path_text);
    return result;
}

jayess_value *jayess_std_path_dirname(jayess_value *path) {
    char *path_text = jayess_value_stringify(path);
    const char *last;
    jayess_value *result;
    if (path_text == NULL || path_text[0] == '\0') {
        free(path_text);
        return jayess_value_from_string(".");
    }
    last = jayess_path_last_separator(path_text);
    if (last == NULL) {
        free(path_text);
        return jayess_value_from_string(".");
    }
    if (last == path_text) {
        path_text[1] = '\0';
    } else {
        path_text[last - path_text] = '\0';
    }
    result = jayess_value_from_string(path_text);
    free(path_text);
    return result;
}

jayess_value *jayess_std_path_extname(jayess_value *path) {
    char *path_text = jayess_value_stringify(path);
    const char *last_sep;
    const char *last_dot;
    jayess_value *result;
    if (path_text == NULL) {
        return jayess_value_from_string("");
    }
    last_sep = jayess_path_last_separator(path_text);
    last_dot = strrchr(path_text, '.');
    if (last_dot == NULL || (last_sep != NULL && last_dot < last_sep + 1)) {
        result = jayess_value_from_string("");
    } else {
        result = jayess_value_from_string(last_dot);
    }
    free(path_text);
    return result;
}

static char *jayess_substring(const char *text, size_t start, size_t end) {
    size_t len;
    char *out;
    if (text == NULL || end < start) {
        return jayess_strdup("");
    }
    len = end - start;
    out = (char *)malloc(len + 1);
    if (out == NULL) {
        return jayess_strdup("");
    }
    memcpy(out, text + start, len);
    out[len] = '\0';
    return out;
}

static int jayess_hex_value(char ch) {
    if (ch >= '0' && ch <= '9') {
        return ch - '0';
    }
    if (ch >= 'a' && ch <= 'f') {
        return ch - 'a' + 10;
    }
    if (ch >= 'A' && ch <= 'F') {
        return ch - 'A' + 10;
    }
    return -1;
}

static char *jayess_percent_decode(const char *text) {
    size_t len = text != NULL ? strlen(text) : 0;
    char *out = (char *)malloc(len + 1);
    size_t i;
    size_t j = 0;
    if (out == NULL) {
        return jayess_strdup("");
    }
    for (i = 0; i < len; i++) {
        if (text[i] == '%' && i + 2 < len) {
            int hi = jayess_hex_value(text[i + 1]);
            int lo = jayess_hex_value(text[i + 2]);
            if (hi >= 0 && lo >= 0) {
                out[j++] = (char)((hi << 4) | lo);
                i += 2;
                continue;
            }
        }
        out[j++] = text[i] == '+' ? ' ' : text[i];
    }
    out[j] = '\0';
    return out;
}

static int jayess_url_should_encode(unsigned char ch) {
    return !(isalnum(ch) || ch == '-' || ch == '_' || ch == '.' || ch == '~');
}

static char *jayess_percent_encode(const char *text) {
    static const char *hex = "0123456789ABCDEF";
    size_t len = text != NULL ? strlen(text) : 0;
    size_t out_len = 0;
    size_t i;
    size_t j = 0;
    char *out;
    for (i = 0; i < len; i++) {
        out_len += jayess_url_should_encode((unsigned char)text[i]) ? 3 : 1;
    }
    out = (char *)malloc(out_len + 1);
    if (out == NULL) {
        return jayess_strdup("");
    }
    for (i = 0; i < len; i++) {
        unsigned char ch = (unsigned char)text[i];
        if (jayess_url_should_encode(ch)) {
            out[j++] = '%';
            out[j++] = hex[(ch >> 4) & 15];
            out[j++] = hex[ch & 15];
        } else {
            out[j++] = (char)ch;
        }
    }
    out[j] = '\0';
    return out;
}

static char *jayess_http_trim_copy(const char *text) {
    const char *start = text != NULL ? text : "";
    const char *end = start + strlen(start);
    while (start < end && isspace((unsigned char)*start)) {
        start++;
    }
    while (end > start && isspace((unsigned char)*(end - 1))) {
        end--;
    }
    return jayess_substring(start, 0, (size_t)(end - start));
}

static const char *jayess_http_line_end(const char *cursor) {
    while (cursor != NULL && *cursor != '\0' && *cursor != '\r' && *cursor != '\n') {
        cursor++;
    }
    return cursor;
}

static const char *jayess_http_next_line(const char *cursor) {
    if (cursor == NULL) {
        return NULL;
    }
    if (*cursor == '\r' && *(cursor + 1) == '\n') {
        return cursor + 2;
    }
    if (*cursor == '\r' || *cursor == '\n') {
        return cursor + 1;
    }
    return cursor;
}

static const char *jayess_http_header_boundary(const char *text) {
    const char *cursor = text != NULL ? text : "";
    while (*cursor != '\0') {
        if (cursor[0] == '\r' && cursor[1] == '\n' && cursor[2] == '\r' && cursor[3] == '\n') {
            return cursor;
        }
        if (cursor[0] == '\n' && cursor[1] == '\n') {
            return cursor;
        }
        cursor++;
    }
    return NULL;
}

static jayess_object *jayess_http_parse_header_object(const char *text) {
    jayess_object *headers = jayess_object_new();
    const char *cursor = text != NULL ? text : "";
    while (*cursor != '\0') {
        const char *line_end = jayess_http_line_end(cursor);
        const char *colon = cursor;
        while (colon < line_end && *colon != ':') {
            colon++;
        }
        if (colon < line_end) {
            char *key_raw = jayess_substring(cursor, 0, (size_t)(colon - cursor));
            char *value_raw = jayess_substring(colon + 1, 0, (size_t)(line_end - colon - 1));
            char *key = jayess_http_trim_copy(key_raw);
            char *value = jayess_http_trim_copy(value_raw);
            if (key != NULL && key[0] != '\0') {
                jayess_object_set_value(headers, key, jayess_value_from_string(value != NULL ? value : ""));
            }
            free(key_raw);
            free(value_raw);
            free(key);
            free(value);
        }
        cursor = jayess_http_next_line(line_end);
    }
    return headers;
}

static int jayess_http_text_contains_ci(const char *text, const char *token) {
    size_t text_len = text != NULL ? strlen(text) : 0;
    size_t token_len = token != NULL ? strlen(token) : 0;
    size_t i;
    if (token_len == 0 || text_len < token_len) {
        return 0;
    }
    for (i = 0; i + token_len <= text_len; i++) {
        size_t j = 0;
        while (j < token_len && tolower((unsigned char)text[i + j]) == tolower((unsigned char)token[j])) {
            j++;
        }
        if (j == token_len) {
            return 1;
        }
    }
    return 0;
}

static int jayess_http_text_eq_ci(const char *left, const char *right) {
    size_t i = 0;
    if (left == NULL || right == NULL) {
        return left == right;
    }
    while (left[i] != '\0' && right[i] != '\0') {
        if (tolower((unsigned char)left[i]) != tolower((unsigned char)right[i])) {
            return 0;
        }
        i++;
    }
    return left[i] == '\0' && right[i] == '\0';
}

static int jayess_http_is_redirect_status(int status) {
    return status == 301 || status == 302 || status == 303 || status == 307 || status == 308;
}

static char *jayess_http_request_current_url(jayess_object *request_object) {
    char *scheme_text = request_object != NULL ? jayess_value_stringify(jayess_object_get(request_object, "scheme")) : jayess_strdup("http");
    char *host_text = request_object != NULL ? jayess_value_stringify(jayess_object_get(request_object, "host")) : jayess_strdup("");
    char *path_text = request_object != NULL ? jayess_value_stringify(jayess_object_get(request_object, "path")) : jayess_strdup("/");
    int port = (int)jayess_value_to_number(request_object != NULL ? jayess_object_get(request_object, "port") : jayess_value_from_number(80));
    size_t total;
    char *url;
    const char *scheme = scheme_text != NULL && scheme_text[0] != '\0' ? scheme_text : "http";
    int default_port = strcmp(scheme, "https") == 0 ? 443 : 80;
    if (host_text == NULL || host_text[0] == '\0') {
        free(scheme_text);
        free(host_text);
        free(path_text);
        return jayess_strdup("");
    }
    total = strlen(scheme) + strlen(host_text) + strlen(path_text != NULL && path_text[0] != '\0' ? path_text : "/") + 32;
    url = (char *)malloc(total);
    if (url == NULL) {
        free(scheme_text);
        free(host_text);
        free(path_text);
        return jayess_strdup("");
    }
    if (port > 0 && port != default_port) {
        snprintf(url, total, "%s://%s:%d%s", scheme, host_text, port, path_text != NULL && path_text[0] != '\0' ? path_text : "/");
    } else {
        snprintf(url, total, "%s://%s%s", scheme, host_text, path_text != NULL && path_text[0] != '\0' ? path_text : "/");
    }
    free(scheme_text);
    free(host_text);
    free(path_text);
    return url;
}

static int jayess_std_socket_configure_timeout(jayess_socket_handle handle, int timeout) {
    if (timeout < 0) {
        timeout = 0;
    }
    if (handle == JAYESS_INVALID_SOCKET) {
        return 0;
    }
#ifdef _WIN32
    {
        DWORD timeout_value = (DWORD)timeout;
        return setsockopt(handle, SOL_SOCKET, SO_RCVTIMEO, (const char *)&timeout_value, sizeof(timeout_value)) == 0 &&
            setsockopt(handle, SOL_SOCKET, SO_SNDTIMEO, (const char *)&timeout_value, sizeof(timeout_value)) == 0;
    }
#else
    {
        struct timeval timeout_value;
        timeout_value.tv_sec = timeout / 1000;
        timeout_value.tv_usec = (timeout % 1000) * 1000;
        return setsockopt(handle, SOL_SOCKET, SO_RCVTIMEO, &timeout_value, sizeof(timeout_value)) == 0 &&
            setsockopt(handle, SOL_SOCKET, SO_SNDTIMEO, &timeout_value, sizeof(timeout_value)) == 0;
    }
#endif
}

static jayess_value *jayess_http_header_get_ci(jayess_object *headers, const char *key) {
    jayess_object_entry *entry = headers != NULL ? headers->head : NULL;
    while (entry != NULL) {
        if (entry->key != NULL && jayess_http_text_eq_ci(entry->key, key)) {
            return entry->value;
        }
        entry = entry->next;
    }
    return NULL;
}

static int jayess_http_headers_transfer_chunked(jayess_object *headers) {
    jayess_value *value = jayess_http_header_get_ci(headers, "Transfer-Encoding");
    if (value != NULL) {
        char *text = jayess_value_stringify(value);
        int matches = jayess_http_text_contains_ci(text, "chunked");
        free(text);
        if (matches) {
            return 1;
        }
    }
    return 0;
}

static int jayess_http_header_value_contains_ci(jayess_object *headers, const char *key, const char *needle) {
    jayess_value *value = jayess_http_header_get_ci(headers, key);
    if (value != NULL) {
        char *text = jayess_value_stringify(value);
        int matches = jayess_http_text_contains_ci(text, needle);
        free(text);
        if (matches) {
            return 1;
        }
    }
    return 0;
}

static long jayess_http_headers_content_length(jayess_object *headers) {
    jayess_value *value = jayess_http_header_get_ci(headers, "Content-Length");
    if (value != NULL) {
        char *text = jayess_value_stringify(value);
        char *trimmed = jayess_http_trim_copy(text);
        char *end_ptr;
        long length = -1;
        if (trimmed != NULL && trimmed[0] != '\0') {
            length = strtol(trimmed, &end_ptr, 10);
            if (end_ptr == trimmed || *end_ptr != '\0' || length < 0) {
                length = -1;
            }
        }
        free(text);
        free(trimmed);
        return length;
    }
    return -1;
}

static char *jayess_http_decode_chunked_body(const char *body) {
    const char *cursor = body != NULL ? body : "";
    char *out = jayess_strdup("");
    size_t out_len = 0;
    if (out == NULL) {
        return jayess_strdup("");
    }
    while (*cursor != '\0') {
        const char *line_end = jayess_http_line_end(cursor);
        const char *size_end = cursor;
        size_t chunk_size = 0;
        char *size_raw;
        char *size_text;
        char *end_ptr;
        char *next;
        if (line_end == cursor) {
            break;
        }
        while (size_end < line_end && *size_end != ';') {
            size_end++;
        }
        size_raw = jayess_substring(cursor, 0, (size_t)(size_end - cursor));
        size_text = jayess_http_trim_copy(size_raw);
        free(size_raw);
        if (size_text == NULL || size_text[0] == '\0') {
            free(size_text);
            free(out);
            return jayess_strdup(body != NULL ? body : "");
        }
        chunk_size = (size_t)strtoul(size_text, &end_ptr, 16);
        if (end_ptr == size_text || *end_ptr != '\0') {
            free(size_text);
            free(out);
            return jayess_strdup(body != NULL ? body : "");
        }
        free(size_text);
        cursor = jayess_http_next_line(line_end);
        if (chunk_size == 0) {
            return out;
        }
        if (strlen(cursor) < chunk_size) {
            free(out);
            return jayess_strdup(body != NULL ? body : "");
        }
        next = (char *)realloc(out, out_len + chunk_size + 1);
        if (next == NULL) {
            free(out);
            return jayess_strdup("");
        }
        out = next;
        memcpy(out + out_len, cursor, chunk_size);
        out_len += chunk_size;
        out[out_len] = '\0';
        cursor += chunk_size;
        if (cursor[0] == '\r' && cursor[1] == '\n') {
            cursor += 2;
        } else if (cursor[0] == '\n') {
            cursor += 1;
        } else if (cursor[0] != '\0') {
            free(out);
            return jayess_strdup(body != NULL ? body : "");
        }
    }
    return out;
}

static int jayess_http_chunked_body_complete(const char *body, size_t available) {
    const char *cursor = body != NULL ? body : "";
    const char *end = cursor + available;
    while (cursor < end) {
        const char *line_end = cursor;
        const char *size_end = cursor;
        char *size_raw;
        char *size_text;
        char *end_ptr;
        size_t chunk_size;
        while (line_end < end && *line_end != '\r' && *line_end != '\n') {
            line_end++;
        }
        if (line_end >= end) {
            return 0;
        }
        while (size_end < line_end && *size_end != ';') {
            size_end++;
        }
        size_raw = jayess_substring(cursor, 0, (size_t)(size_end - cursor));
        size_text = jayess_http_trim_copy(size_raw);
        free(size_raw);
        if (size_text == NULL || size_text[0] == '\0') {
            free(size_text);
            return 0;
        }
        chunk_size = (size_t)strtoul(size_text, &end_ptr, 16);
        free(size_text);
        if (end_ptr == NULL || *end_ptr != '\0') {
            return 0;
        }
        cursor = jayess_http_next_line(line_end);
        if ((size_t)(end - cursor) < chunk_size) {
            return 0;
        }
        cursor += chunk_size;
        if (cursor >= end) {
            return 0;
        }
        if (cursor[0] == '\r') {
            if (cursor + 1 >= end || cursor[1] != '\n') {
                return 0;
            }
            cursor += 2;
        } else if (cursor[0] == '\n') {
            cursor += 1;
        } else {
            return 0;
        }
        if (chunk_size == 0) {
            if (cursor < end && cursor[0] == '\r') {
                return cursor + 1 < end && cursor[1] == '\n';
            }
            if (cursor < end && cursor[0] == '\n') {
                return 1;
            }
            return cursor >= end;
        }
    }
    return 0;
}

static void jayess_http_body_stream_mark_ended(jayess_value *env) {
    if (env != NULL && env->kind == JAYESS_VALUE_OBJECT && env->as.object_value != NULL) {
        jayess_object_set_value(env->as.object_value, "readableEnded", jayess_value_from_bool(1));
        jayess_object_set_value(env->as.object_value, "closed", jayess_value_from_bool(1));
    }
}

static jayess_value *jayess_http_body_stream_socket_value(jayess_value *env) {
    if (env == NULL || env->kind != JAYESS_VALUE_OBJECT || env->as.object_value == NULL) {
        return NULL;
    }
    return jayess_object_get(env->as.object_value, "__jayess_http_body_socket");
}

static void jayess_http_body_stream_close_socket(jayess_value *env) {
    jayess_value *socket_value = jayess_http_body_stream_socket_value(env);
    jayess_socket_handle handle;
    if (socket_value != NULL && socket_value->kind == JAYESS_VALUE_OBJECT && jayess_std_kind_is(socket_value, "Socket")) {
        jayess_std_socket_close_method(socket_value);
        if (env != NULL && env->kind == JAYESS_VALUE_OBJECT && env->as.object_value != NULL) {
            jayess_object_set_value(env->as.object_value, "__jayess_http_body_socket", jayess_value_undefined());
        }
        return;
    }
    handle = jayess_std_socket_handle(env);
    if (handle != JAYESS_INVALID_SOCKET) {
        jayess_std_socket_close_handle(handle);
        jayess_std_socket_set_handle(env, JAYESS_INVALID_SOCKET);
    }
}

static void jayess_http_body_stream_close_native(jayess_value *env) {
#ifdef _WIN32
    if (env != NULL && env->kind == JAYESS_VALUE_OBJECT && env->as.object_value != NULL && env->as.object_value->native_handle != NULL) {
        jayess_winhttp_stream_state *state = (jayess_winhttp_stream_state *)env->as.object_value->native_handle;
        if (state->request != NULL) {
            WinHttpCloseHandle(state->request);
        }
        if (state->connection != NULL) {
            WinHttpCloseHandle(state->connection);
        }
        if (state->session != NULL) {
            WinHttpCloseHandle(state->session);
        }
        free(state);
        env->as.object_value->native_handle = NULL;
    }
#else
    (void)env;
#endif
}

static void jayess_http_body_stream_emit_end(jayess_value *env) {
    jayess_http_body_stream_mark_ended(env);
    jayess_http_body_stream_close_socket(env);
    jayess_http_body_stream_close_native(env);
    if (env != NULL && env->kind == JAYESS_VALUE_OBJECT && env->as.object_value != NULL) {
        jayess_std_stream_emit(env, "end", jayess_value_undefined());
    }
}

static jayess_array *jayess_http_body_stream_prebuffer_bytes(jayess_value *env) {
    jayess_value *prebuffer;
    if (env == NULL || env->kind != JAYESS_VALUE_OBJECT || env->as.object_value == NULL) {
        return NULL;
    }
    prebuffer = jayess_object_get(env->as.object_value, "__jayess_http_body_prebuffer");
    if (prebuffer == NULL || prebuffer->kind != JAYESS_VALUE_OBJECT || !jayess_std_kind_is(prebuffer, "Uint8Array")) {
        return NULL;
    }
    return jayess_std_bytes_slot(prebuffer);
}

static int jayess_http_body_stream_take_prebuffer(jayess_value *env, unsigned char *buffer, int max_count) {
    jayess_array *bytes = jayess_http_body_stream_prebuffer_bytes(env);
    int offset;
    int available;
    int count;
    int i;
    if (bytes == NULL || max_count <= 0 || env == NULL || env->kind != JAYESS_VALUE_OBJECT || env->as.object_value == NULL) {
        return 0;
    }
    offset = (int)jayess_value_to_number(jayess_object_get(env->as.object_value, "__jayess_http_body_prebuffer_offset"));
    if (offset < 0) {
        offset = 0;
    }
    if (offset >= bytes->count) {
        return 0;
    }
    available = bytes->count - offset;
    count = available < max_count ? available : max_count;
    for (i = 0; i < count; i++) {
        buffer[i] = (unsigned char)((int)jayess_value_to_number(jayess_array_get(bytes, offset + i)) & 255);
    }
    jayess_object_set_value(env->as.object_value, "__jayess_http_body_prebuffer_offset", jayess_value_from_number((double)(offset + count)));
    return count;
}

static int jayess_http_body_stream_read_raw(jayess_value *env, unsigned char *buffer, int max_count) {
    int count = 0;
    if (env == NULL || env->kind != JAYESS_VALUE_OBJECT || env->as.object_value == NULL || buffer == NULL || max_count <= 0) {
        return -1;
    }
    count = jayess_http_body_stream_take_prebuffer(env, buffer, max_count);
    if (count > 0) {
        return count;
    }
    {
        jayess_value *socket_value = jayess_http_body_stream_socket_value(env);
        if (socket_value != NULL && socket_value->kind == JAYESS_VALUE_OBJECT && jayess_std_kind_is(socket_value, "Socket")) {
            int did_timeout = 0;
            if (jayess_std_tls_state(socket_value) != NULL) {
                return jayess_std_tls_read_bytes(socket_value, buffer, max_count, &did_timeout);
            }
        }
        jayess_socket_handle handle = jayess_std_socket_handle(env);
        if (handle != JAYESS_INVALID_SOCKET) {
            int read_count = (int)recv(handle, (char *)buffer, max_count, 0);
            if (read_count < 0) {
                jayess_std_stream_emit_error(env, "failed to read from HTTP body stream");
                jayess_http_body_stream_close_socket(env);
                jayess_http_body_stream_close_native(env);
                jayess_http_body_stream_mark_ended(env);
                return -1;
            }
            if (read_count == 0) {
                jayess_http_body_stream_close_socket(env);
                jayess_http_body_stream_close_native(env);
                return 0;
            }
            return read_count;
        }
#ifdef _WIN32
        if (env->as.object_value->native_handle != NULL) {
            jayess_winhttp_stream_state *state = (jayess_winhttp_stream_state *)env->as.object_value->native_handle;
            DWORD available = 0;
            DWORD read_now = 0;
            DWORD to_read = 0;
            if (state == NULL || state->request == NULL) {
                return 0;
            }
            if (!WinHttpQueryDataAvailable(state->request, &available)) {
                jayess_std_stream_emit_error(env, "failed to query HTTPS body availability");
                jayess_http_body_stream_close_native(env);
                jayess_http_body_stream_mark_ended(env);
                return -1;
            }
            if (available == 0) {
                jayess_http_body_stream_close_native(env);
                return 0;
            }
            to_read = available < (DWORD)max_count ? available : (DWORD)max_count;
            if (!WinHttpReadData(state->request, buffer, to_read, &read_now)) {
                jayess_std_stream_emit_error(env, "failed to read from HTTPS body stream");
                jayess_http_body_stream_close_native(env);
                jayess_http_body_stream_mark_ended(env);
                return -1;
            }
            if (read_now == 0) {
                jayess_http_body_stream_close_native(env);
                return 0;
            }
            return (int)read_now;
        }
#endif
        return 0;
    }
}

static int jayess_http_body_stream_read_byte(jayess_value *env, unsigned char *out) {
    unsigned char byte_value = 0;
    int read_count = jayess_http_body_stream_read_raw(env, &byte_value, 1);
    if (read_count > 0 && out != NULL) {
        *out = byte_value;
    }
    return read_count;
}

static char *jayess_http_body_stream_read_line(jayess_value *env) {
    size_t capacity = 32;
    size_t length = 0;
    char *line = (char *)malloc(capacity);
    if (line == NULL) {
        return NULL;
    }
    for (;;) {
        unsigned char byte_value = 0;
        int read_count = jayess_http_body_stream_read_byte(env, &byte_value);
        if (read_count <= 0) {
            free(line);
            return NULL;
        }
        if (byte_value == '\n') {
            if (length > 0 && line[length - 1] == '\r') {
                length--;
            }
            line[length] = '\0';
            return line;
        }
        if (length + 1 >= capacity) {
            size_t next_capacity = capacity * 2;
            char *next = (char *)realloc(line, next_capacity);
            if (next == NULL) {
                free(line);
                return NULL;
            }
            line = next;
            capacity = next_capacity;
        }
        line[length++] = (char)byte_value;
    }
}

static int jayess_http_body_stream_consume_crlf(jayess_value *env) {
    unsigned char first = 0;
    int read_first = jayess_http_body_stream_read_byte(env, &first);
    if (read_first <= 0) {
        return 0;
    }
    if (first == '\n') {
        return 1;
    }
    if (first == '\r') {
        unsigned char second = 0;
        int read_second = jayess_http_body_stream_read_byte(env, &second);
        return read_second > 0 && second == '\n';
    }
    return 0;
}

static jayess_value *jayess_http_body_stream_make_string(const unsigned char *buffer, int count) {
    char *text;
    jayess_value *result;
    if (buffer == NULL || count <= 0) {
        return jayess_value_from_string("");
    }
    text = (char *)malloc((size_t)count + 1);
    if (text == NULL) {
        return jayess_value_undefined();
    }
    memcpy(text, buffer, (size_t)count);
    text[count] = '\0';
    result = jayess_value_from_string(text);
    free(text);
    return result;
}

static jayess_value *jayess_http_body_stream_read_non_chunked(jayess_value *env, jayess_value *size_value, int as_bytes) {
    int requested = jayess_std_stream_requested_size(size_value, 4095);
    long remaining = (long)jayess_value_to_number(jayess_object_get(env->as.object_value, "__jayess_http_body_remaining"));
    unsigned char *buffer;
    int total = 0;
    if (remaining == 0) {
        jayess_http_body_stream_emit_end(env);
        return jayess_value_null();
    }
    if (remaining > 0 && requested > remaining) {
        requested = (int)remaining;
    }
    buffer = (unsigned char *)malloc((size_t)requested);
    if (buffer == NULL) {
        jayess_std_stream_emit_error(env, "failed to allocate HTTP body buffer");
        return jayess_value_undefined();
    }
    while (total < requested) {
        int read_count = jayess_http_body_stream_read_raw(env, buffer + total, requested - total);
        if (read_count < 0) {
            free(buffer);
            return jayess_value_undefined();
        }
        if (read_count == 0) {
            if (remaining < 0) {
                break;
            }
            jayess_std_stream_emit_error(env, "HTTP body stream ended before expected Content-Length");
            free(buffer);
            return jayess_value_undefined();
        }
        total += read_count;
        if (remaining < 0) {
            break;
        }
    }
    if (total == 0) {
        free(buffer);
        jayess_http_body_stream_emit_end(env);
        return jayess_value_null();
    }
    if (remaining > 0) {
        remaining -= total;
        if (remaining < 0) {
            remaining = 0;
        }
        jayess_object_set_value(env->as.object_value, "__jayess_http_body_remaining", jayess_value_from_number((double)remaining));
        if (remaining == 0) {
            jayess_http_body_stream_mark_ended(env);
            jayess_http_body_stream_close_socket(env);
        }
    }
    if (as_bytes) {
        jayess_value *result = jayess_std_uint8_array_from_bytes(buffer, (size_t)total);
        free(buffer);
        if (remaining == 0) {
            jayess_std_stream_emit(env, "end", jayess_value_undefined());
        }
        return result;
    }
    {
        jayess_value *result = jayess_http_body_stream_make_string(buffer, total);
        free(buffer);
        if (remaining == 0) {
            jayess_std_stream_emit(env, "end", jayess_value_undefined());
        }
        return result;
    }
}

static jayess_value *jayess_http_body_stream_read_chunked(jayess_value *env, jayess_value *size_value, int as_bytes) {
    int requested = jayess_std_stream_requested_size(size_value, 4095);
    unsigned char *buffer = (unsigned char *)malloc((size_t)requested);
    int total = 0;
    if (buffer == NULL) {
        jayess_std_stream_emit_error(env, "failed to allocate HTTP chunk buffer");
        return jayess_value_undefined();
    }
    for (;;) {
        long chunk_remaining = (long)jayess_value_to_number(jayess_object_get(env->as.object_value, "__jayess_http_body_chunk_remaining"));
        if (jayess_value_as_bool(jayess_object_get(env->as.object_value, "__jayess_http_body_chunk_finished"))) {
            free(buffer);
            jayess_http_body_stream_emit_end(env);
            return jayess_value_null();
        }
        if (chunk_remaining < 0) {
            char *line = jayess_http_body_stream_read_line(env);
            char *trimmed;
            char *end_ptr;
            unsigned long chunk_size;
            if (line == NULL) {
                free(buffer);
                jayess_std_stream_emit_error(env, "failed to read HTTP chunk size");
                return jayess_value_undefined();
            }
            trimmed = jayess_http_trim_copy(line);
            free(line);
            if (trimmed == NULL || trimmed[0] == '\0') {
                free(trimmed);
                continue;
            }
            {
                char *semi = strchr(trimmed, ';');
                if (semi != NULL) {
                    *semi = '\0';
                }
            }
            chunk_size = strtoul(trimmed, &end_ptr, 16);
            free(trimmed);
            if (end_ptr == NULL || *end_ptr != '\0') {
                free(buffer);
                jayess_std_stream_emit_error(env, "invalid HTTP chunk size");
                return jayess_value_undefined();
            }
            if (chunk_size == 0) {
                for (;;) {
                    char *trailer = jayess_http_body_stream_read_line(env);
                    if (trailer == NULL) {
                        free(buffer);
                        jayess_std_stream_emit_error(env, "failed to read HTTP chunk trailer");
                        return jayess_value_undefined();
                    }
                    if (trailer[0] == '\0') {
                        free(trailer);
                        break;
                    }
                    free(trailer);
                }
                jayess_object_set_value(env->as.object_value, "__jayess_http_body_chunk_finished", jayess_value_from_bool(1));
                if (total == 0) {
                    free(buffer);
                    jayess_http_body_stream_emit_end(env);
                    return jayess_value_null();
                }
                break;
            }
            chunk_remaining = (long)chunk_size;
            jayess_object_set_value(env->as.object_value, "__jayess_http_body_chunk_remaining", jayess_value_from_number((double)chunk_remaining));
        }
        if (chunk_remaining > 0) {
            int need = requested - total;
            int take = (int)chunk_remaining;
            while (need > 0 && take > 0) {
                int read_target = need < take ? need : take;
                int read_count = jayess_http_body_stream_read_raw(env, buffer + total, read_target);
                if (read_count <= 0) {
                    free(buffer);
                    jayess_std_stream_emit_error(env, "HTTP chunk body ended unexpectedly");
                    return jayess_value_undefined();
                }
                total += read_count;
                need -= read_count;
                take -= read_count;
                chunk_remaining -= read_count;
                jayess_object_set_value(env->as.object_value, "__jayess_http_body_chunk_remaining", jayess_value_from_number((double)chunk_remaining));
                if (total >= requested) {
                    break;
                }
            }
            if (chunk_remaining == 0) {
                if (!jayess_http_body_stream_consume_crlf(env)) {
                    free(buffer);
                    jayess_std_stream_emit_error(env, "invalid HTTP chunk terminator");
                    return jayess_value_undefined();
                }
                jayess_object_set_value(env->as.object_value, "__jayess_http_body_chunk_remaining", jayess_value_from_number(-1));
            }
            if (total > 0) {
                break;
            }
        }
    }
    if (total == 0) {
        free(buffer);
        return jayess_value_null();
    }
    if (as_bytes) {
        jayess_value *result = jayess_std_uint8_array_from_bytes(buffer, (size_t)total);
        free(buffer);
        return result;
    }
    {
        jayess_value *result = jayess_http_body_stream_make_string(buffer, total);
        free(buffer);
        return result;
    }
}

static jayess_value *jayess_http_body_stream_read_chunk(jayess_value *env, jayess_value *size_value, int as_bytes) {
    if (env == NULL || env->kind != JAYESS_VALUE_OBJECT || env->as.object_value == NULL) {
        return jayess_value_undefined();
    }
    if (jayess_value_as_bool(jayess_object_get(env->as.object_value, "readableEnded"))) {
        return jayess_value_null();
    }
    if (jayess_value_as_bool(jayess_object_get(env->as.object_value, "__jayess_http_body_chunked"))) {
        return jayess_http_body_stream_read_chunked(env, size_value, as_bytes);
    }
    return jayess_http_body_stream_read_non_chunked(env, size_value, as_bytes);
}

static jayess_value *jayess_http_body_stream_new(jayess_socket_handle handle, const unsigned char *prebuffer, size_t prebuffer_len, jayess_object *headers) {
    jayess_object *object;
    long content_length;
    int chunked;
    if (handle == JAYESS_INVALID_SOCKET) {
        return jayess_value_undefined();
    }
    object = jayess_object_new();
    if (object == NULL) {
        jayess_std_socket_close_handle(handle);
        return jayess_value_undefined();
    }
    object->socket_handle = handle;
    chunked = jayess_http_headers_transfer_chunked(headers);
    content_length = jayess_http_headers_content_length(headers);
    jayess_object_set_value(object, "__jayess_std_kind", jayess_value_from_string("HttpBodyStream"));
    jayess_object_set_value(object, "closed", jayess_value_from_bool(0));
    jayess_object_set_value(object, "readableEnded", jayess_value_from_bool(0));
    jayess_object_set_value(object, "errored", jayess_value_from_bool(0));
    jayess_object_set_value(object, "__jayess_http_body_chunked", jayess_value_from_bool(chunked));
    jayess_object_set_value(object, "__jayess_http_body_remaining", jayess_value_from_number(chunked ? -1 : (double)content_length));
    jayess_object_set_value(object, "__jayess_http_body_chunk_remaining", jayess_value_from_number(-1));
    jayess_object_set_value(object, "__jayess_http_body_chunk_finished", jayess_value_from_bool(0));
    jayess_object_set_value(object, "__jayess_http_body_prebuffer", jayess_std_uint8_array_from_bytes(prebuffer != NULL ? prebuffer : (const unsigned char *)"", prebuffer_len));
    jayess_object_set_value(object, "__jayess_http_body_prebuffer_offset", jayess_value_from_number(0));
    if (!chunked && content_length == 0 && prebuffer_len == 0) {
        jayess_value *stream_value = jayess_value_from_object(object);
        jayess_http_body_stream_mark_ended(stream_value);
        jayess_http_body_stream_close_socket(stream_value);
        return stream_value;
    }
    return jayess_value_from_object(object);
}

static jayess_value *jayess_http_body_stream_new_from_socket(jayess_value *socket_value, const unsigned char *prebuffer, size_t prebuffer_len, jayess_object *headers) {
    jayess_value *stream_value = jayess_http_body_stream_new(jayess_std_socket_handle(socket_value), prebuffer, prebuffer_len, headers);
    if (stream_value != NULL && stream_value->kind == JAYESS_VALUE_OBJECT && stream_value->as.object_value != NULL) {
        jayess_object_set_value(stream_value->as.object_value, "__jayess_http_body_socket", socket_value != NULL ? socket_value : jayess_value_undefined());
    }
    return stream_value;
}

static int jayess_http_response_complete(const char *buffer, size_t length) {
    const char *header_end;
    const char *body_start;
    size_t header_bytes;
    size_t body_bytes;
    char *headers_text;
    jayess_object *headers;
    long content_length;
    int chunked;
    if (buffer == NULL || length == 0) {
        return 0;
    }
    header_end = jayess_http_header_boundary(buffer);
    if (header_end == NULL) {
        return 0;
    }
    body_start = (header_end[0] == '\r' && header_end[1] == '\n') ? header_end + 4 : header_end + 2;
    header_bytes = (size_t)(body_start - buffer);
    if (length < header_bytes) {
        return 0;
    }
    body_bytes = length - header_bytes;
    {
        const char *line_end = jayess_http_line_end(buffer);
        const char *header_start = jayess_http_next_line(line_end);
        headers_text = jayess_substring(header_start, 0, (size_t)(header_end - header_start));
    }
    headers = jayess_http_parse_header_object(headers_text);
    free(headers_text);
    chunked = jayess_http_headers_transfer_chunked(headers);
    if (chunked) {
        return jayess_http_chunked_body_complete(body_start, body_bytes);
    }
    content_length = jayess_http_headers_content_length(headers);
    if (content_length >= 0) {
        return body_bytes >= (size_t)content_length;
    }
    return 0;
}

static char *jayess_http_format_header_lines(jayess_object *headers) {
    jayess_object_entry *entry = headers != NULL ? headers->head : NULL;
    char *out = jayess_strdup("");
    while (entry != NULL) {
        char *value = jayess_value_stringify(entry->value);
        size_t current_len = strlen(out != NULL ? out : "");
        size_t key_len = strlen(entry->key != NULL ? entry->key : "");
        size_t value_len = strlen(value != NULL ? value : "");
        char *next = (char *)malloc(current_len + key_len + value_len + 5);
        if (next == NULL) {
            free(value);
            break;
        }
        sprintf(next, "%s%s: %s\r\n", out != NULL ? out : "", entry->key != NULL ? entry->key : "", value != NULL ? value : "");
        free(out);
        out = next;
        free(value);
        entry = entry->next;
    }
    return out;
}

static int jayess_http_socket_read_raw(jayess_value *socket_value, unsigned char *buffer, int max_count, int *did_timeout) {
    jayess_socket_handle handle;
    if (did_timeout != NULL) {
        *did_timeout = 0;
    }
    if (socket_value == NULL || socket_value->kind != JAYESS_VALUE_OBJECT || !jayess_std_kind_is(socket_value, "Socket")) {
        return -1;
    }
    if (jayess_std_tls_state(socket_value) != NULL) {
        return jayess_std_tls_read_bytes(socket_value, buffer, max_count, did_timeout);
    }
    handle = jayess_std_socket_handle(socket_value);
    if (handle == JAYESS_INVALID_SOCKET) {
        return 0;
    }
    return (int)recv(handle, (char *)buffer, max_count, 0);
}

static char *jayess_http_read_all_socket_value(jayess_value *socket_value) {
    size_t capacity = 1024;
    size_t length = 0;
    char *buffer = (char *)malloc(capacity + 1);
    if (buffer == NULL) {
        return jayess_strdup("");
    }
    for (;;) {
        unsigned char chunk[1024];
        int read_count = jayess_http_socket_read_raw(socket_value, chunk, (int)sizeof(chunk), NULL);
        if (read_count <= 0) {
            break;
        }
        if (length + (size_t)read_count >= capacity) {
            size_t next_capacity = capacity;
            while (length + (size_t)read_count >= next_capacity) {
                next_capacity *= 2;
            }
            {
                char *next = (char *)realloc(buffer, next_capacity + 1);
                if (next == NULL) {
                    break;
                }
                buffer = next;
                capacity = next_capacity;
            }
        }
        memcpy(buffer + length, chunk, (size_t)read_count);
        length += (size_t)read_count;
        buffer[length] = '\0';
        if (jayess_http_response_complete(buffer, length)) {
            break;
        }
    }
    buffer[length] = '\0';
    return buffer;
}

static char *jayess_http_read_all_socket(jayess_socket_handle handle) {
    jayess_value *socket_value = jayess_std_socket_value_from_handle(handle, "", 0);
    if (socket_value == NULL) {
        return jayess_strdup("");
    }
    return jayess_http_read_all_socket_value(socket_value);
}

static jayess_value *jayess_http_read_response_stream_socket(jayess_value *socket_value) {
    size_t capacity = 1024;
    size_t length = 0;
    char *buffer = (char *)malloc(capacity + 1);
    const char *header_end;
    const char *line_end;
    const char *sp1;
    const char *sp2;
    const char *header_start;
    const char *body_start;
    char *version;
    char *status_text;
    char *reason;
    char *headers_text;
    jayess_object *headers;
    jayess_object *result;
    jayess_value *body_stream;
    double status_number;
    size_t body_len;
    if (buffer == NULL) {
        return jayess_value_undefined();
    }
    for (;;) {
        unsigned char chunk[1024];
        int read_count = jayess_http_socket_read_raw(socket_value, chunk, (int)sizeof(chunk), NULL);
        if (read_count <= 0) {
            free(buffer);
            return jayess_value_undefined();
        }
        if (length + (size_t)read_count >= capacity) {
            size_t next_capacity = capacity;
            while (length + (size_t)read_count >= next_capacity) {
                next_capacity *= 2;
            }
            {
                char *next = (char *)realloc(buffer, next_capacity + 1);
                if (next == NULL) {
                    free(buffer);
                    return jayess_value_undefined();
                }
                buffer = next;
                capacity = next_capacity;
            }
        }
        memcpy(buffer + length, chunk, (size_t)read_count);
        length += (size_t)read_count;
        buffer[length] = '\0';
        header_end = jayess_http_header_boundary(buffer);
        if (header_end != NULL) {
            break;
        }
    }
    header_end = jayess_http_header_boundary(buffer);
    if (header_end == NULL) {
        free(buffer);
        return jayess_value_undefined();
    }
    line_end = jayess_http_line_end(buffer);
    sp1 = buffer;
    while (sp1 < line_end && *sp1 != ' ') {
        sp1++;
    }
    if (sp1 >= line_end) {
        free(buffer);
        return jayess_value_undefined();
    }
    sp2 = sp1 + 1;
    while (sp2 < line_end && *sp2 != ' ') {
        sp2++;
    }
    version = jayess_substring(buffer, 0, (size_t)(sp1 - buffer));
    status_text = jayess_substring(sp1 + 1, 0, (size_t)((sp2 < line_end ? sp2 : line_end) - sp1 - 1));
    reason = sp2 < line_end ? jayess_substring(sp2 + 1, 0, (size_t)(line_end - sp2 - 1)) : jayess_strdup("");
    header_start = jayess_http_next_line(line_end);
    headers_text = jayess_substring(header_start, 0, (size_t)(header_end - header_start));
    headers = jayess_http_parse_header_object(headers_text);
    body_start = (header_end[0] == '\r' && header_end[1] == '\n') ? header_end + 4 : header_end + 2;
    body_len = length >= (size_t)(body_start - buffer) ? length - (size_t)(body_start - buffer) : 0;
    body_stream = jayess_http_body_stream_new_from_socket(socket_value, (const unsigned char *)body_start, body_len, headers);
    status_number = atof(status_text != NULL ? status_text : "0");
    result = jayess_object_new();
    jayess_object_set_value(result, "version", jayess_value_from_string(version));
    jayess_object_set_value(result, "status", jayess_value_from_number(status_number));
    jayess_object_set_value(result, "reason", jayess_value_from_string(reason));
    jayess_object_set_value(result, "statusText", jayess_value_from_string(reason));
    jayess_object_set_value(result, "ok", jayess_value_from_bool(status_number >= 200.0 && status_number < 300.0));
    jayess_object_set_value(result, "headers", jayess_value_from_object(headers));
    jayess_object_set_value(result, "bodyStream", body_stream);
    free(version);
    free(status_text);
    free(reason);
    free(headers_text);
    free(buffer);
    return jayess_value_from_object(result);
}

static jayess_value *jayess_http_read_response_stream(jayess_socket_handle handle) {
    jayess_value *socket_value = jayess_std_socket_value_from_handle(handle, "", 0);
    if (socket_value == NULL) {
        return jayess_value_undefined();
    }
    return jayess_http_read_response_stream_socket(socket_value);
}

static void jayess_http_split_host_port(const char *input, char **host_out, int *port_out, int default_port) {
    const char *value = input != NULL ? input : "";
    const char *last_colon = strrchr(value, ':');
    if (host_out != NULL) {
        *host_out = NULL;
    }
    if (port_out != NULL) {
        *port_out = default_port;
    }
    if (last_colon != NULL && strchr(last_colon + 1, ':') == NULL) {
        char *host = jayess_substring(value, 0, (size_t)(last_colon - value));
        int port = atoi(last_colon + 1);
        if (host_out != NULL) {
            *host_out = host;
        } else {
            free(host);
        }
        if (port_out != NULL && port > 0) {
            *port_out = port;
        }
        return;
    }
    if (host_out != NULL) {
        *host_out = jayess_strdup(value);
    }
}

static jayess_object *jayess_http_request_object_from_url_value(jayess_value *input, const char *default_method) {
    jayess_value *parsed = jayess_std_url_parse(input);
    jayess_object *parsed_object = jayess_value_as_object(parsed);
    char *protocol = parsed_object != NULL ? jayess_value_stringify(jayess_object_get(parsed_object, "protocol")) : jayess_strdup("");
    char *host_raw = parsed_object != NULL ? jayess_value_stringify(jayess_object_get(parsed_object, "host")) : jayess_strdup("");
    char *host = NULL;
    int port = protocol != NULL && strcmp(protocol, "https:") == 0 ? 443 : 80;
    char *pathname = parsed_object != NULL ? jayess_value_stringify(jayess_object_get(parsed_object, "pathname")) : jayess_strdup("/");
    char *query = parsed_object != NULL ? jayess_value_stringify(jayess_object_get(parsed_object, "query")) : jayess_strdup("");
    char *full_path;
    jayess_object *request_object = jayess_object_new();
    size_t path_len;
    if (protocol != NULL && strcmp(protocol, "https:") == 0) {
        jayess_http_split_host_port(host_raw, &host, &port, 443);
    } else {
        jayess_http_split_host_port(host_raw, &host, &port, 80);
    }
    path_len = strlen(pathname != NULL && pathname[0] != '\0' ? pathname : "/") + strlen(query != NULL && query[0] != '\0' ? query : "") + 2;
    full_path = (char *)malloc(path_len);
    if (full_path == NULL) {
        free(protocol);
        free(host_raw);
        free(host);
        free(pathname);
        free(query);
        return NULL;
    }
    sprintf(full_path, "%s%s%s", pathname != NULL && pathname[0] != '\0' ? pathname : "/", query != NULL && query[0] != '\0' ? "?" : "", query != NULL ? query : "");
    jayess_object_set_value(request_object, "method", jayess_value_from_string(default_method != NULL && default_method[0] != '\0' ? default_method : "GET"));
    jayess_object_set_value(request_object, "path", jayess_value_from_string(full_path));
    jayess_object_set_value(request_object, "scheme", jayess_value_from_string(protocol != NULL && strcmp(protocol, "https:") == 0 ? "https" : "http"));
    jayess_object_set_value(request_object, "version", jayess_value_from_string("HTTP/1.1"));
    jayess_object_set_value(request_object, "headers", jayess_value_from_object(jayess_object_new()));
    jayess_object_set_value(request_object, "body", jayess_value_from_string(""));
    jayess_object_set_value(request_object, "host", jayess_value_from_string(host != NULL ? host : ""));
    jayess_object_set_value(request_object, "port", jayess_value_from_number((double)port));
    free(protocol);
    free(host_raw);
    free(host);
    free(pathname);
    free(query);
    free(full_path);
    return request_object;
}

static void jayess_http_prepare_request_headers(jayess_object *request_object, const char *host_text, int port) {
    jayess_object *headers;
    char *body_text;
    char *host_header;
    char body_len_text[32];
    if (request_object == NULL) {
        return;
    }
    headers = jayess_value_as_object(jayess_object_get(request_object, "headers"));
    if (headers == NULL) {
        headers = jayess_object_new();
        jayess_object_set_value(request_object, "headers", jayess_value_from_object(headers));
    }
    if (jayess_http_header_get_ci(headers, "Host") == NULL) {
        jayess_object_set_value(headers, "Host", jayess_value_from_string(host_text != NULL ? host_text : ""));
    }
    if (jayess_http_header_get_ci(headers, "Connection") == NULL) {
        jayess_object_set_value(headers, "Connection", jayess_value_from_string("close"));
    }
    if (jayess_http_header_get_ci(headers, "Content-Length") == NULL && !jayess_http_headers_transfer_chunked(headers)) {
        body_text = jayess_value_stringify(jayess_object_get(request_object, "body"));
        if (body_text != NULL && body_text[0] != '\0') {
            snprintf(body_len_text, sizeof(body_len_text), "%u", (unsigned int)strlen(body_text));
            jayess_object_set_value(headers, "Content-Length", jayess_value_from_string(body_len_text));
        }
        free(body_text);
    }
}

static jayess_value *jayess_http_request_from_parts(jayess_object *request_object) {
    int redirects = 0;
    int max_redirects = (int)jayess_value_to_number(jayess_object_get(request_object, "maxRedirects"));
    if (max_redirects < 0) {
        max_redirects = 0;
    }
    if (max_redirects == 0 && jayess_object_get(request_object, "maxRedirects") == NULL) {
        max_redirects = 5;
    }
    for (;;) {
        char *host_text = jayess_value_stringify(jayess_object_get(request_object, "host"));
        int port = (int)jayess_value_to_number(jayess_object_get(request_object, "port"));
        int timeout = (int)jayess_value_to_number(jayess_object_get(request_object, "timeout"));
        jayess_value *request_text_value;
        char *request_text;
        char port_text[32];
        struct addrinfo hints;
        struct addrinfo *results = NULL;
        struct addrinfo *entry;
        jayess_socket_handle handle = JAYESS_INVALID_SOCKET;
        int status;
        jayess_value *response;
        jayess_object *response_object;
        int response_status;
        jayess_value *location_value;
        char *location_text;

        if (host_text == NULL || host_text[0] == '\0' || port <= 0 || !jayess_std_socket_runtime_ready()) {
            free(host_text);
            return jayess_value_undefined();
        }

        jayess_http_prepare_request_headers(request_object, host_text, port);
        request_text_value = jayess_std_http_format_request(jayess_value_from_object(request_object));
        request_text = jayess_value_stringify(request_text_value);

        if (request_text == NULL) {
            free(host_text);
            return jayess_value_undefined();
        }

        snprintf(port_text, sizeof(port_text), "%d", port);
        memset(&hints, 0, sizeof(hints));
        hints.ai_family = AF_UNSPEC;
        hints.ai_socktype = SOCK_STREAM;
        status = getaddrinfo(host_text, port_text, &hints, &results);
        if (status != 0 || results == NULL) {
            free(request_text);
            free(host_text);
            return jayess_value_undefined();
        }

        for (entry = results; entry != NULL; entry = entry->ai_next) {
            handle = socket(entry->ai_family, entry->ai_socktype, entry->ai_protocol);
            if (handle == JAYESS_INVALID_SOCKET) {
                continue;
            }
            if (connect(handle, entry->ai_addr, (int)entry->ai_addrlen) == 0) {
                break;
            }
            jayess_std_socket_close_handle(handle);
            handle = JAYESS_INVALID_SOCKET;
        }
        freeaddrinfo(results);
        if (handle == JAYESS_INVALID_SOCKET) {
            free(request_text);
            free(host_text);
            return jayess_value_undefined();
        }
        if (timeout > 0 && !jayess_std_socket_configure_timeout(handle, timeout)) {
            jayess_std_socket_close_handle(handle);
            free(request_text);
            free(host_text);
            return jayess_value_undefined();
        }

        {
            size_t length = strlen(request_text);
            size_t offset = 0;
            while (offset < length) {
                int sent = (int)send(handle, request_text + offset, (int)(length - offset), 0);
                if (sent <= 0) {
                    jayess_std_socket_close_handle(handle);
                    free(request_text);
                    free(host_text);
                    return jayess_value_undefined();
                }
                offset += (size_t)sent;
            }
        }

#ifdef _WIN32
        shutdown(handle, SD_SEND);
#else
        shutdown(handle, SHUT_WR);
#endif
        {
            char *response_text = jayess_http_read_all_socket(handle);
            response = jayess_std_http_parse_response(jayess_value_from_string(response_text));
            free(response_text);
        }
        jayess_std_socket_close_handle(handle);
        free(request_text);
        free(host_text);

        response_object = jayess_value_as_object(response);
        if (response_object == NULL) {
            return response;
        }
        response_status = (int)jayess_value_to_number(jayess_object_get(response_object, "status"));
        if (!jayess_http_is_redirect_status(response_status) || redirects >= max_redirects) {
            char *final_url = jayess_http_request_current_url(request_object);
            jayess_object_set_value(response_object, "redirected", jayess_value_from_bool(redirects > 0));
            jayess_object_set_value(response_object, "redirectCount", jayess_value_from_number((double)redirects));
            jayess_object_set_value(response_object, "url", jayess_value_from_string(final_url != NULL ? final_url : ""));
            free(final_url);
            return response;
        }
        location_value = jayess_http_header_get_ci(jayess_value_as_object(jayess_object_get(response_object, "headers")), "Location");
        location_text = jayess_value_stringify(location_value);
        if (location_text == NULL || location_text[0] == '\0') {
            free(location_text);
            return response;
        }
        if (strncmp(location_text, "http://", 7) == 0) {
            jayess_object *redirect_object = jayess_http_request_object_from_url_value(jayess_value_from_string(location_text), jayess_value_as_string(jayess_object_get(request_object, "method")));
            if (redirect_object == NULL) {
                free(location_text);
                return response;
            }
            jayess_object_set_value(redirect_object, "timeout", jayess_object_get(request_object, "timeout"));
            request_object = redirect_object;
        } else if (location_text[0] == '/') {
            jayess_object_set_value(request_object, "path", jayess_value_from_string(location_text));
        } else {
            free(location_text);
            return response;
        }
        if (response_status == 301 || response_status == 302 || response_status == 303) {
            jayess_object_set_value(request_object, "method", jayess_value_from_string("GET"));
            jayess_object_set_value(request_object, "body", jayess_value_from_string(""));
        }
        jayess_object_set_value(request_object, "headers", jayess_value_from_object(jayess_object_new()));
        jayess_object_set_value(request_object, "maxRedirects", jayess_value_from_number((double)max_redirects));
        redirects++;
        free(location_text);
    }
}

static jayess_value *jayess_http_request_stream_from_parts(jayess_object *request_object) {
    int redirects = 0;
    int max_redirects = (int)jayess_value_to_number(jayess_object_get(request_object, "maxRedirects"));
    if (max_redirects < 0) {
        max_redirects = 0;
    }
    if (max_redirects == 0 && jayess_object_get(request_object, "maxRedirects") == NULL) {
        max_redirects = 5;
    }
    for (;;) {
        char *host_text = jayess_value_stringify(jayess_object_get(request_object, "host"));
        int port = (int)jayess_value_to_number(jayess_object_get(request_object, "port"));
        int timeout = (int)jayess_value_to_number(jayess_object_get(request_object, "timeout"));
        jayess_value *request_text_value;
        char *request_text;
        char port_text[32];
        struct addrinfo hints;
        struct addrinfo *results = NULL;
        struct addrinfo *entry;
        jayess_socket_handle handle = JAYESS_INVALID_SOCKET;
        int status;
        jayess_value *response;
        jayess_object *response_object;
        int response_status;
        jayess_value *location_value;
        char *location_text;

        if (host_text == NULL || host_text[0] == '\0' || port <= 0 || !jayess_std_socket_runtime_ready()) {
            free(host_text);
            return jayess_value_undefined();
        }

        jayess_http_prepare_request_headers(request_object, host_text, port);
        request_text_value = jayess_std_http_format_request(jayess_value_from_object(request_object));
        request_text = jayess_value_stringify(request_text_value);
        if (request_text == NULL) {
            free(host_text);
            return jayess_value_undefined();
        }

        snprintf(port_text, sizeof(port_text), "%d", port);
        memset(&hints, 0, sizeof(hints));
        hints.ai_family = AF_UNSPEC;
        hints.ai_socktype = SOCK_STREAM;
        status = getaddrinfo(host_text, port_text, &hints, &results);
        if (status != 0 || results == NULL) {
            free(request_text);
            free(host_text);
            return jayess_value_undefined();
        }
        for (entry = results; entry != NULL; entry = entry->ai_next) {
            handle = socket(entry->ai_family, entry->ai_socktype, entry->ai_protocol);
            if (handle == JAYESS_INVALID_SOCKET) {
                continue;
            }
            if (connect(handle, entry->ai_addr, (int)entry->ai_addrlen) == 0) {
                break;
            }
            jayess_std_socket_close_handle(handle);
            handle = JAYESS_INVALID_SOCKET;
        }
        freeaddrinfo(results);
        if (handle == JAYESS_INVALID_SOCKET) {
            free(request_text);
            free(host_text);
            return jayess_value_undefined();
        }
        if (timeout > 0 && !jayess_std_socket_configure_timeout(handle, timeout)) {
            jayess_std_socket_close_handle(handle);
            free(request_text);
            free(host_text);
            return jayess_value_undefined();
        }
        {
            size_t length = strlen(request_text);
            size_t offset = 0;
            while (offset < length) {
                int sent = (int)send(handle, request_text + offset, (int)(length - offset), 0);
                if (sent <= 0) {
                    jayess_std_socket_close_handle(handle);
                    free(request_text);
                    free(host_text);
                    return jayess_value_undefined();
                }
                offset += (size_t)sent;
            }
        }
#ifdef _WIN32
        shutdown(handle, SD_SEND);
#else
        shutdown(handle, SHUT_WR);
#endif
        response = jayess_http_read_response_stream(handle);
        free(request_text);
        free(host_text);
        response_object = jayess_value_as_object(response);
        if (response_object == NULL) {
            jayess_std_socket_close_handle(handle);
            return response;
        }
        response_status = (int)jayess_value_to_number(jayess_object_get(response_object, "status"));
        if (!jayess_http_is_redirect_status(response_status) || redirects >= max_redirects) {
            char *final_url = jayess_http_request_current_url(request_object);
            jayess_object_set_value(response_object, "redirected", jayess_value_from_bool(redirects > 0));
            jayess_object_set_value(response_object, "redirectCount", jayess_value_from_number((double)redirects));
            jayess_object_set_value(response_object, "url", jayess_value_from_string(final_url != NULL ? final_url : ""));
            free(final_url);
            return response;
        }
        location_value = jayess_http_header_get_ci(jayess_value_as_object(jayess_object_get(response_object, "headers")), "Location");
        location_text = jayess_value_stringify(location_value);
        {
            jayess_value *body_stream = jayess_object_get(response_object, "bodyStream");
            if (body_stream != NULL && body_stream->kind == JAYESS_VALUE_OBJECT && jayess_std_kind_is(body_stream, "HttpBodyStream")) {
                jayess_std_http_body_stream_close_method(body_stream);
            } else {
                jayess_std_socket_close_handle(handle);
            }
        }
        if (location_text == NULL || location_text[0] == '\0') {
            free(location_text);
            return response;
        }
        if (strncmp(location_text, "http://", 7) == 0) {
            jayess_object *redirect_object = jayess_http_request_object_from_url_value(jayess_value_from_string(location_text), jayess_value_as_string(jayess_object_get(request_object, "method")));
            if (redirect_object == NULL) {
                free(location_text);
                return response;
            }
            jayess_object_set_value(redirect_object, "timeout", jayess_object_get(request_object, "timeout"));
            request_object = redirect_object;
        } else if (location_text[0] == '/') {
            jayess_object_set_value(request_object, "path", jayess_value_from_string(location_text));
        } else {
            free(location_text);
            return response;
        }
        if (response_status == 301 || response_status == 302 || response_status == 303) {
            jayess_object_set_value(request_object, "method", jayess_value_from_string("GET"));
            jayess_object_set_value(request_object, "body", jayess_value_from_string(""));
        }
        jayess_object_set_value(request_object, "headers", jayess_value_from_object(jayess_object_new()));
        jayess_object_set_value(request_object, "maxRedirects", jayess_value_from_number((double)max_redirects));
        redirects++;
        free(location_text);
    }
}

static jayess_value *jayess_https_request_via_tls_from_parts(jayess_object *request_object, int stream_response) {
    int redirects = 0;
    int max_redirects = (int)jayess_value_to_number(jayess_object_get(request_object, "maxRedirects"));
    jayess_array *http11_alpn;
    if (max_redirects < 0) {
        max_redirects = 0;
    }
    if (max_redirects == 0 && jayess_object_get(request_object, "maxRedirects") == NULL) {
        max_redirects = 5;
    }
    http11_alpn = jayess_array_new();
    if (http11_alpn != NULL) {
        jayess_array_push_value(http11_alpn, jayess_value_from_string("http/1.1"));
    }
    for (;;) {
        char *host_text = jayess_value_stringify(jayess_object_get(request_object, "host"));
        int port = (int)jayess_value_to_number(jayess_object_get(request_object, "port"));
        int timeout = (int)jayess_value_to_number(jayess_object_get(request_object, "timeout"));
        jayess_value *reject_value = jayess_object_get(request_object, "rejectUnauthorized");
        jayess_object *tls_options;
        jayess_value *socket_value;
        jayess_value *request_text_value;
        char *request_text;
        jayess_value *response;
        jayess_object *response_object;
        int response_status;
        jayess_value *location_value;
        char *location_text;
        if (host_text == NULL || host_text[0] == '\0' || port <= 0 || !jayess_std_socket_runtime_ready()) {
            free(host_text);
            return jayess_value_undefined();
        }
        jayess_http_prepare_request_headers(request_object, host_text, port);
        tls_options = jayess_object_new();
        jayess_object_set_value(tls_options, "host", jayess_value_from_string(host_text));
        jayess_object_set_value(tls_options, "port", jayess_value_from_number((double)port));
        jayess_object_set_value(tls_options, "rejectUnauthorized", reject_value != NULL ? reject_value : jayess_value_from_bool(1));
        jayess_object_set_value(tls_options, "timeout", jayess_value_from_number((double)timeout));
        if (http11_alpn != NULL) {
            jayess_object_set_value(tls_options, "alpnProtocols", jayess_value_from_array(http11_alpn));
        }
        jayess_std_https_copy_tls_request_settings(tls_options, request_object);
        socket_value = jayess_std_tls_connect(jayess_value_from_object(tls_options));
        if (jayess_has_exception()) {
            free(host_text);
            return jayess_value_undefined();
        }
        if (socket_value == NULL || socket_value->kind != JAYESS_VALUE_OBJECT || !jayess_std_kind_is(socket_value, "Socket")) {
            free(host_text);
            return jayess_value_undefined();
        }
        request_text_value = jayess_std_http_format_request(jayess_value_from_object(request_object));
        request_text = jayess_value_stringify(request_text_value);
        if (request_text == NULL) {
            jayess_std_socket_close_method(socket_value);
            free(host_text);
            return jayess_value_undefined();
        }
        if (!jayess_value_as_bool(jayess_std_socket_write_method(socket_value, jayess_value_from_string(request_text)))) {
            free(request_text);
            jayess_std_socket_close_method(socket_value);
            free(host_text);
            return jayess_value_undefined();
        }
        free(request_text);
        if (stream_response) {
            response = jayess_http_read_response_stream_socket(socket_value);
        } else {
            char *response_text = jayess_http_read_all_socket_value(socket_value);
            response = jayess_std_http_parse_response(jayess_value_from_string(response_text != NULL ? response_text : ""));
            free(response_text);
        }
        if (!stream_response) {
            jayess_std_socket_close_method(socket_value);
        }
        free(host_text);
        response_object = jayess_value_as_object(response);
        if (response_object == NULL) {
            return response;
        }
        response_status = (int)jayess_value_to_number(jayess_object_get(response_object, "status"));
        if (!jayess_http_is_redirect_status(response_status) || redirects >= max_redirects) {
            char *final_url = jayess_http_request_current_url(request_object);
            jayess_object_set_value(response_object, "redirected", jayess_value_from_bool(redirects > 0));
            jayess_object_set_value(response_object, "redirectCount", jayess_value_from_number((double)redirects));
            jayess_object_set_value(response_object, "url", jayess_value_from_string(final_url != NULL ? final_url : ""));
            free(final_url);
            return response;
        }
        location_value = jayess_http_header_get_ci(jayess_value_as_object(jayess_object_get(response_object, "headers")), "Location");
        location_text = jayess_value_stringify(location_value);
        if (stream_response) {
            jayess_value *body_stream = jayess_object_get(response_object, "bodyStream");
            if (body_stream != NULL && body_stream->kind == JAYESS_VALUE_OBJECT && jayess_std_kind_is(body_stream, "HttpBodyStream")) {
                jayess_std_http_body_stream_close_method(body_stream);
            }
        }
        if (location_text == NULL || location_text[0] == '\0') {
            free(location_text);
            return response;
        }
        if (strncmp(location_text, "https://", 8) == 0 || strncmp(location_text, "http://", 7) == 0) {
            jayess_object *redirect_object = jayess_http_request_object_from_url_value(jayess_value_from_string(location_text), jayess_value_as_string(jayess_object_get(request_object, "method")));
            if (redirect_object == NULL) {
                free(location_text);
                return response;
            }
            jayess_object_set_value(redirect_object, "timeout", jayess_object_get(request_object, "timeout"));
            jayess_object_set_value(redirect_object, "rejectUnauthorized", jayess_object_get(request_object, "rejectUnauthorized"));
            jayess_std_https_copy_tls_request_settings(redirect_object, request_object);
            request_object = redirect_object;
        } else if (location_text[0] == '/') {
            jayess_object_set_value(request_object, "path", jayess_value_from_string(location_text));
        } else {
            free(location_text);
            return response;
        }
        if (response_status == 301 || response_status == 302 || response_status == 303) {
            jayess_object_set_value(request_object, "method", jayess_value_from_string("GET"));
            jayess_object_set_value(request_object, "body", jayess_value_from_string(""));
        }
        jayess_object_set_value(request_object, "headers", jayess_value_from_object(jayess_object_new()));
        jayess_object_set_value(request_object, "maxRedirects", jayess_value_from_number((double)max_redirects));
        redirects++;
        free(location_text);
    }
}

jayess_value *jayess_std_querystring_parse(jayess_value *input) {
    char *text = jayess_value_stringify(input);
    const char *cursor = text != NULL ? text : "";
    jayess_object *object = jayess_object_new();
    while (*cursor != '\0') {
        const char *part_start = cursor;
        const char *part_end;
        const char *eq;
        char *key_raw;
        char *value_raw;
        char *key;
        char *value;
        while (*cursor != '\0' && *cursor != '&') {
            cursor++;
        }
        part_end = cursor;
        eq = part_start;
        while (eq < part_end && *eq != '=') {
            eq++;
        }
        key_raw = jayess_substring(part_start, 0, (size_t)(eq - part_start));
        value_raw = eq < part_end ? jayess_substring(eq + 1, 0, (size_t)(part_end - eq - 1)) : jayess_strdup("");
        key = jayess_percent_decode(key_raw);
        value = jayess_percent_decode(value_raw);
        if (key != NULL && key[0] != '\0') {
            jayess_object_set_value(object, key, jayess_value_from_string(value != NULL ? value : ""));
        }
        free(key_raw);
        free(value_raw);
        free(key);
        free(value);
        if (*cursor == '&') {
            cursor++;
        }
    }
    free(text);
    return jayess_value_from_object(object);
}

jayess_value *jayess_std_querystring_stringify(jayess_value *parts) {
    jayess_object *object = jayess_value_as_object(parts);
    jayess_object_entry *entry = object != NULL ? object->head : NULL;
    char *out = jayess_strdup("");
    size_t out_len = 0;
    int first = 1;
    while (entry != NULL) {
        char *key = jayess_percent_encode(entry->key);
        char *value_text = jayess_value_stringify(entry->value);
        char *value = jayess_percent_encode(value_text != NULL ? value_text : "");
        size_t key_len = strlen(key != NULL ? key : "");
        size_t value_len = strlen(value != NULL ? value : "");
        char *next = (char *)malloc(out_len + key_len + value_len + (first ? 2 : 3));
        if (next == NULL) {
            free(key);
            free(value_text);
            free(value);
            break;
        }
        sprintf(next, "%s%s%s=%s", out != NULL ? out : "", first ? "" : "&", key != NULL ? key : "", value != NULL ? value : "");
        free(out);
        out = next;
        out_len = strlen(out);
        first = 0;
        free(key);
        free(value_text);
        free(value);
        entry = entry->next;
    }
    {
        jayess_value *result = jayess_value_from_string(out != NULL ? out : "");
        free(out);
        return result;
    }
}

jayess_value *jayess_std_url_parse(jayess_value *input) {
    char *text = jayess_value_stringify(input);
    const char *value = text != NULL ? text : "";
    const char *scheme = strstr(value, "://");
    const char *after_scheme = scheme != NULL ? scheme + 3 : value;
    const char *path_start = strchr(after_scheme, '/');
    const char *query_start = strchr(after_scheme, '?');
    const char *hash_start = strchr(after_scheme, '#');
    const char *host_end = after_scheme + strlen(after_scheme);
    const char *path_end;
    const char *query_end;
    char *protocol;
    char *host;
    char *pathname;
    char *query;
    char *hash;
    jayess_object *object = jayess_object_new();
    if (path_start != NULL && path_start < host_end) {
        host_end = path_start;
    }
    if (query_start != NULL && query_start < host_end) {
        host_end = query_start;
    }
    if (hash_start != NULL && hash_start < host_end) {
        host_end = hash_start;
    }
    path_end = value + strlen(value);
    if (query_start != NULL && query_start < path_end) {
        path_end = query_start;
    }
    if (hash_start != NULL && hash_start < path_end) {
        path_end = hash_start;
    }
    query_end = hash_start != NULL ? hash_start : value + strlen(value);
    protocol = scheme != NULL ? jayess_substring(value, 0, (size_t)(scheme - value + 1)) : jayess_strdup("");
    host = jayess_substring(after_scheme, 0, (size_t)(host_end - after_scheme));
    pathname = path_start != NULL ? jayess_substring(path_start, 0, (size_t)(path_end - path_start)) : jayess_strdup("");
    query = query_start != NULL ? jayess_substring(query_start + 1, 0, (size_t)(query_end - query_start - 1)) : jayess_strdup("");
    hash = hash_start != NULL ? jayess_strdup(hash_start) : jayess_strdup("");
    jayess_object_set_value(object, "href", jayess_value_from_string(value));
    jayess_object_set_value(object, "protocol", jayess_value_from_string(protocol));
    jayess_object_set_value(object, "host", jayess_value_from_string(host));
    jayess_object_set_value(object, "pathname", jayess_value_from_string(pathname));
    jayess_object_set_value(object, "query", jayess_value_from_string(query));
    jayess_object_set_value(object, "hash", jayess_value_from_string(hash));
    jayess_object_set_value(object, "queryObject", jayess_std_querystring_parse(jayess_value_from_string(query)));
    free(protocol);
    free(host);
    free(pathname);
    free(query);
    free(hash);
    free(text);
    return jayess_value_from_object(object);
}

jayess_value *jayess_std_url_format(jayess_value *parts) {
    jayess_object *object = jayess_value_as_object(parts);
    char *protocol = jayess_strdup(object != NULL ? jayess_value_as_string(jayess_object_get(object, "protocol")) : "");
    char *host = jayess_strdup(object != NULL ? jayess_value_as_string(jayess_object_get(object, "host")) : "");
    char *pathname = jayess_strdup(object != NULL ? jayess_value_as_string(jayess_object_get(object, "pathname")) : "");
    char *query = jayess_strdup(object != NULL ? jayess_value_as_string(jayess_object_get(object, "query")) : "");
    char *hash = jayess_strdup(object != NULL ? jayess_value_as_string(jayess_object_get(object, "hash")) : "");
    size_t len = strlen(protocol != NULL ? protocol : "") + strlen(host != NULL ? host : "") + strlen(pathname != NULL ? pathname : "") + strlen(query != NULL ? query : "") + strlen(hash != NULL ? hash : "") + 8;
    char *out = (char *)malloc(len);
    if (out == NULL) {
        return jayess_value_from_string("");
    }
    out[0] = '\0';
    if (protocol != NULL && protocol[0] != '\0') {
        strcat(out, protocol);
        if (strstr(protocol, "://") == NULL) {
            strcat(out, "//");
        }
    }
    strcat(out, host != NULL ? host : "");
    strcat(out, pathname != NULL ? pathname : "");
    if (query != NULL && query[0] != '\0') {
        strcat(out, "?");
        strcat(out, query);
    }
    if (hash != NULL && hash[0] != '\0') {
        if (hash[0] != '#') {
            strcat(out, "#");
        }
        strcat(out, hash);
    }
    {
        jayess_value *result = jayess_value_from_string(out);
        free(protocol);
        free(host);
        free(pathname);
        free(query);
        free(hash);
        free(out);
        return result;
    }
}

jayess_value *jayess_std_http_parse_request(jayess_value *input) {
    char *text = jayess_value_stringify(input);
    const char *value = text != NULL ? text : "";
    const char *line_end = jayess_http_line_end(value);
    const char *sp1 = value;
    const char *sp2;
    const char *header_start;
    const char *header_end;
    const char *body_start;
    char *method;
    char *path;
    char *version;
    char *headers_text;
    char *body;
    jayess_object *result;
    if (line_end == value) {
        free(text);
        return jayess_value_undefined();
    }
    while (sp1 < line_end && *sp1 != ' ') {
        sp1++;
    }
    if (sp1 >= line_end) {
        free(text);
        return jayess_value_undefined();
    }
    sp2 = sp1 + 1;
    while (sp2 < line_end && *sp2 != ' ') {
        sp2++;
    }
    if (sp2 >= line_end) {
        free(text);
        return jayess_value_undefined();
    }
    method = jayess_substring(value, 0, (size_t)(sp1 - value));
    path = jayess_substring(sp1 + 1, 0, (size_t)(sp2 - sp1 - 1));
    version = jayess_substring(sp2 + 1, 0, (size_t)(line_end - sp2 - 1));
    header_start = jayess_http_next_line(line_end);
    header_end = jayess_http_header_boundary(header_start);
    body_start = header_end != NULL ? ((header_end[0] == '\r' && header_end[1] == '\n') ? header_end + 4 : header_end + 2) : value + strlen(value);
    headers_text = jayess_substring(header_start, 0, (size_t)(header_end != NULL ? (size_t)(header_end - header_start) : 0));
    body = jayess_strdup(body_start != NULL ? body_start : "");
    result = jayess_object_new();
    jayess_object_set_value(result, "method", jayess_value_from_string(method));
    jayess_object_set_value(result, "path", jayess_value_from_string(path));
    jayess_object_set_value(result, "version", jayess_value_from_string(version));
    jayess_object_set_value(result, "headers", jayess_value_from_object(jayess_http_parse_header_object(headers_text)));
    jayess_object_set_value(result, "body", jayess_value_from_string(body));
    free(method);
    free(path);
    free(version);
    free(headers_text);
    free(body);
    free(text);
    return jayess_value_from_object(result);
}

jayess_value *jayess_std_http_format_request(jayess_value *parts) {
    jayess_object *object = jayess_value_as_object(parts);
    char *method = object != NULL ? jayess_value_stringify(jayess_object_get(object, "method")) : jayess_strdup("GET");
    char *path = object != NULL ? jayess_value_stringify(jayess_object_get(object, "path")) : jayess_strdup("/");
    char *version = object != NULL ? jayess_value_stringify(jayess_object_get(object, "version")) : jayess_strdup("HTTP/1.1");
    jayess_object *headers = object != NULL ? jayess_value_as_object(jayess_object_get(object, "headers")) : NULL;
    char *headers_text = jayess_http_format_header_lines(headers);
    char *body = object != NULL ? jayess_value_stringify(jayess_object_get(object, "body")) : jayess_strdup("");
    size_t total = strlen(method != NULL ? method : "") + strlen(path != NULL ? path : "") + strlen(version != NULL ? version : "") + strlen(headers_text != NULL ? headers_text : "") + strlen(body != NULL ? body : "") + 8;
    char *out = (char *)malloc(total);
    jayess_value *result;
    if (out == NULL) {
        free(method);
        free(path);
        free(version);
        free(headers_text);
        free(body);
        return jayess_value_from_string("");
    }
    sprintf(out, "%s %s %s\r\n%s\r\n%s", method != NULL && method[0] != '\0' ? method : "GET", path != NULL && path[0] != '\0' ? path : "/", version != NULL && version[0] != '\0' ? version : "HTTP/1.1", headers_text != NULL ? headers_text : "", body != NULL ? body : "");
    result = jayess_value_from_string(out);
    free(method);
    free(path);
    free(version);
    free(headers_text);
    free(body);
    free(out);
    return result;
}

jayess_value *jayess_std_http_parse_response(jayess_value *input) {
    char *text = jayess_value_stringify(input);
    const char *value = text != NULL ? text : "";
    const char *line_end = jayess_http_line_end(value);
    const char *sp1 = value;
    const char *sp2;
    const char *header_start;
    const char *header_end;
    const char *body_start;
    char *version;
    char *status_text;
    char *reason;
    char *headers_text;
    char *body;
    char *decoded_body;
    jayess_object *headers;
    jayess_object *result;
    double status_number;
    if (line_end == value) {
        free(text);
        return jayess_value_undefined();
    }
    while (sp1 < line_end && *sp1 != ' ') {
        sp1++;
    }
    if (sp1 >= line_end) {
        free(text);
        return jayess_value_undefined();
    }
    sp2 = sp1 + 1;
    while (sp2 < line_end && *sp2 != ' ') {
        sp2++;
    }
    version = jayess_substring(value, 0, (size_t)(sp1 - value));
    status_text = jayess_substring(sp1 + 1, 0, (size_t)((sp2 < line_end ? sp2 : line_end) - sp1 - 1));
    reason = sp2 < line_end ? jayess_substring(sp2 + 1, 0, (size_t)(line_end - sp2 - 1)) : jayess_strdup("");
    header_start = jayess_http_next_line(line_end);
    header_end = jayess_http_header_boundary(header_start);
    body_start = header_end != NULL ? ((header_end[0] == '\r' && header_end[1] == '\n') ? header_end + 4 : header_end + 2) : value + strlen(value);
    headers_text = jayess_substring(header_start, 0, (size_t)(header_end != NULL ? (size_t)(header_end - header_start) : 0));
    body = jayess_strdup(body_start != NULL ? body_start : "");
    headers = jayess_http_parse_header_object(headers_text);
    decoded_body = jayess_http_headers_transfer_chunked(headers) ? jayess_http_decode_chunked_body(body) : jayess_strdup(body != NULL ? body : "");
    status_number = atof(status_text != NULL ? status_text : "0");
    result = jayess_object_new();
    jayess_object_set_value(result, "version", jayess_value_from_string(version));
    jayess_object_set_value(result, "status", jayess_value_from_number(status_number));
    jayess_object_set_value(result, "reason", jayess_value_from_string(reason));
    jayess_object_set_value(result, "statusText", jayess_value_from_string(reason));
    jayess_object_set_value(result, "ok", jayess_value_from_bool(status_number >= 200.0 && status_number < 300.0));
    jayess_object_set_value(result, "headers", jayess_value_from_object(headers));
    jayess_object_set_value(result, "body", jayess_value_from_string(decoded_body != NULL ? decoded_body : ""));
    jayess_object_set_value(result, "bodyBytes", jayess_std_uint8_array_from_bytes((const unsigned char *)(decoded_body != NULL ? decoded_body : ""), decoded_body != NULL ? strlen(decoded_body) : 0));
    free(version);
    free(status_text);
    free(reason);
    free(headers_text);
    free(body);
    free(decoded_body);
    free(text);
    return jayess_value_from_object(result);
}

jayess_value *jayess_std_http_format_response(jayess_value *parts) {
    jayess_object *object = jayess_value_as_object(parts);
    char *version = object != NULL ? jayess_value_stringify(jayess_object_get(object, "version")) : jayess_strdup("HTTP/1.1");
    char *reason = object != NULL ? jayess_value_stringify(jayess_object_get(object, "reason")) : jayess_strdup("OK");
    char *status_text = object != NULL ? jayess_value_stringify(jayess_object_get(object, "status")) : jayess_strdup("200");
    jayess_object *headers = object != NULL ? jayess_value_as_object(jayess_object_get(object, "headers")) : NULL;
    char *headers_text = jayess_http_format_header_lines(headers);
    char *body = object != NULL ? jayess_value_stringify(jayess_object_get(object, "body")) : jayess_strdup("");
    size_t total = strlen(version != NULL ? version : "") + strlen(status_text != NULL ? status_text : "") + strlen(reason != NULL ? reason : "") + strlen(headers_text != NULL ? headers_text : "") + strlen(body != NULL ? body : "") + 8;
    char *out = (char *)malloc(total);
    jayess_value *result;
    if (out == NULL) {
        free(version);
        free(reason);
        free(status_text);
        free(headers_text);
        free(body);
        return jayess_value_from_string("");
    }
    sprintf(out, "%s %s %s\r\n%s\r\n%s", version != NULL && version[0] != '\0' ? version : "HTTP/1.1", status_text != NULL && status_text[0] != '\0' ? status_text : "200", reason != NULL ? reason : "", headers_text != NULL ? headers_text : "", body != NULL ? body : "");
    result = jayess_value_from_string(out);
    free(version);
    free(reason);
    free(status_text);
    free(headers_text);
    free(body);
    free(out);
    return result;
}

static jayess_value *jayess_http_read_request_from_socket(jayess_value *socket_value) {
    char *buffer = NULL;
    size_t buffer_len = 0;
    size_t buffer_cap = 0;
    const char *header_end = NULL;
    jayess_object *headers;
    long content_length = 0;
    size_t total_needed = 0;
    while (1) {
        unsigned char chunk[1024];
        int read_count;
        char *next_buffer;
        read_count = jayess_http_socket_read_raw(socket_value, chunk, (int)sizeof(chunk), NULL);
        if (read_count <= 0) {
            free(buffer);
            return jayess_value_undefined();
        }
        if (buffer_len + (size_t)read_count + 1 > buffer_cap) {
            buffer_cap = (buffer_len + (size_t)read_count + 1) * 2;
            next_buffer = (char *)realloc(buffer, buffer_cap);
            if (next_buffer == NULL) {
                free(buffer);
                return jayess_value_undefined();
            }
            buffer = next_buffer;
        }
        memcpy(buffer + buffer_len, chunk, (size_t)read_count);
        buffer_len += (size_t)read_count;
        buffer[buffer_len] = '\0';
        header_end = jayess_http_header_boundary(buffer);
        if (header_end == NULL) {
            continue;
        }
        {
            const char *line_end = jayess_http_line_end(buffer);
            const char *header_start = jayess_http_next_line(line_end);
            char *headers_text = jayess_substring(header_start, 0, (size_t)(header_end - header_start));
            headers = jayess_http_parse_header_object(headers_text);
            free(headers_text);
        }
        if (jayess_http_headers_transfer_chunked(headers)) {
            const char *body_start = (header_end[0] == '\r' && header_end[1] == '\n') ? header_end + 4 : header_end + 2;
            size_t body_len = buffer_len - (size_t)(body_start - buffer);
            if (jayess_http_chunked_body_complete(body_start, body_len)) {
                break;
            }
            continue;
        }
        content_length = jayess_http_headers_content_length(headers);
        total_needed = ((header_end[0] == '\r' && header_end[1] == '\n') ? (size_t)(header_end - buffer) + 4 : (size_t)(header_end - buffer) + 2) + (content_length > 0 ? (size_t)content_length : 0);
        if (buffer_len >= total_needed) {
            break;
        }
    }
    {
        jayess_value *result = jayess_std_http_parse_request(jayess_value_from_string(buffer != NULL ? buffer : ""));
        free(buffer);
        return result;
    }
}

static int jayess_http_request_wants_keep_alive(jayess_value *request) {
    jayess_object *request_object = jayess_value_as_object(request);
    jayess_object *headers = request_object != NULL ? jayess_value_as_object(jayess_object_get(request_object, "headers")) : NULL;
    const char *version = request_object != NULL ? jayess_value_as_string(jayess_object_get(request_object, "version")) : NULL;
    if (request_object == NULL) {
        return 0;
    }
    if (jayess_http_header_value_contains_ci(headers, "Connection", "close")) {
        return 0;
    }
    if (version != NULL && strcmp(version, "HTTP/1.1") == 0) {
        return 1;
    }
    return jayess_http_header_value_contains_ci(headers, "Connection", "keep-alive");
}

static int jayess_std_http_response_send_headers(jayess_value *env) {
    jayess_http_response_state *state;
    jayess_object *response_object;
    jayess_object *headers;
    jayess_value *status_value;
    jayess_value *headers_value;
    jayess_value *response_text;
    char *response_raw;
    char *header_boundary;
    if (env == NULL || env->kind != JAYESS_VALUE_OBJECT || env->as.object_value == NULL) {
        return 0;
    }
    state = (jayess_http_response_state *)env->as.object_value->native_handle;
    if (state == NULL || state->socket == NULL) {
        return 0;
    }
    if (state->headers_sent) {
        return 1;
    }
    response_object = jayess_object_new();
    status_value = jayess_object_get(env->as.object_value, "statusCode");
    headers_value = jayess_object_get(env->as.object_value, "headers");
    headers = jayess_value_as_object(headers_value);
    if (headers == NULL) {
        headers = jayess_object_new();
        jayess_object_set_value(env->as.object_value, "headers", jayess_value_from_object(headers));
    }
    if (jayess_http_header_get_ci(headers, "Connection") == NULL) {
        jayess_object_set_value(headers, "Connection", jayess_value_from_string(state->keep_alive ? "keep-alive" : "close"));
    }
    if (jayess_http_header_get_ci(headers, "Content-Length") == NULL && !jayess_http_headers_transfer_chunked(headers)) {
        jayess_object_set_value(headers, "Transfer-Encoding", jayess_value_from_string("chunked"));
    }
    state->chunked = jayess_http_headers_transfer_chunked(headers);
    jayess_object_set_value(response_object, "version", jayess_value_from_string("HTTP/1.1"));
    jayess_object_set_value(response_object, "status", status_value != NULL ? status_value : jayess_value_from_number(200));
    jayess_object_set_value(response_object, "reason", jayess_value_from_string("OK"));
    jayess_object_set_value(response_object, "headers", jayess_value_from_object(headers));
    jayess_object_set_value(response_object, "body", jayess_value_from_string(""));
    response_text = jayess_std_http_format_response(jayess_value_from_object(response_object));
    response_raw = jayess_value_stringify(response_text);
    if (response_raw == NULL) {
        return 0;
    }
    header_boundary = strstr(response_raw, "\r\n\r\n");
    if (header_boundary != NULL) {
        header_boundary[4] = '\0';
    }
    if (!jayess_value_as_bool(jayess_std_socket_write_method(state->socket, jayess_value_from_string(response_raw)))) {
        free(response_raw);
        return 0;
    }
    free(response_raw);
    state->headers_sent = 1;
    return 1;
}

static jayess_value *jayess_std_http_response_set_header_method(jayess_value *env, jayess_value *name, jayess_value *value) {
    char *name_text;
    jayess_object *headers;
    if (env == NULL || env->kind != JAYESS_VALUE_OBJECT || env->as.object_value == NULL) {
        return jayess_value_undefined();
    }
    headers = jayess_value_as_object(jayess_object_get(env->as.object_value, "headers"));
    if (headers == NULL) {
        headers = jayess_object_new();
        jayess_object_set_value(env->as.object_value, "headers", jayess_value_from_object(headers));
    }
    name_text = jayess_value_stringify(name);
    if (name_text != NULL && name_text[0] != '\0') {
        jayess_object_set_value(headers, name_text, value != NULL ? value : jayess_value_undefined());
    }
    free(name_text);
    return env;
}

static jayess_value *jayess_std_http_response_write_method(jayess_value *env, jayess_value *chunk) {
    jayess_http_response_state *state;
    char *chunk_text;
    char size_text[32];
    jayess_value *ok;
    if (env == NULL || env->kind != JAYESS_VALUE_OBJECT || env->as.object_value == NULL) {
        return jayess_value_from_bool(0);
    }
    state = (jayess_http_response_state *)env->as.object_value->native_handle;
    if (state == NULL || state->finished) {
        return jayess_value_from_bool(0);
    }
    if (!jayess_std_http_response_send_headers(env)) {
        return jayess_value_from_bool(0);
    }
    if (!state->chunked) {
        return jayess_std_socket_write_method(state->socket, chunk != NULL ? chunk : jayess_value_from_string(""));
    }
    chunk_text = jayess_value_stringify(chunk != NULL ? chunk : jayess_value_from_string(""));
    if (chunk_text == NULL) {
        return jayess_value_from_bool(0);
    }
    if (chunk_text[0] == '\0') {
        free(chunk_text);
        return jayess_value_from_bool(1);
    }
    snprintf(size_text, sizeof(size_text), "%zx\r\n", strlen(chunk_text));
    ok = jayess_std_socket_write_method(state->socket, jayess_value_from_string(size_text));
    if (!jayess_value_as_bool(ok)) {
        free(chunk_text);
        return jayess_value_from_bool(0);
    }
    ok = jayess_std_socket_write_method(state->socket, jayess_value_from_string(chunk_text));
    if (!jayess_value_as_bool(ok)) {
        free(chunk_text);
        return jayess_value_from_bool(0);
    }
    free(chunk_text);
    return jayess_std_socket_write_method(state->socket, jayess_value_from_string("\r\n"));
}

static jayess_value *jayess_std_http_response_end_method(jayess_value *env, jayess_value *chunk) {
    jayess_http_response_state *state;
    jayess_object *headers;
    char *chunk_text;
    char length_text[32];
    if (env == NULL || env->kind != JAYESS_VALUE_OBJECT || env->as.object_value == NULL) {
        return jayess_value_undefined();
    }
    state = (jayess_http_response_state *)env->as.object_value->native_handle;
    if (state == NULL || state->finished) {
        return env;
    }
    headers = jayess_value_as_object(jayess_object_get(env->as.object_value, "headers"));
    if (!state->headers_sent && chunk != NULL && !(chunk->kind == JAYESS_VALUE_UNDEFINED || chunk->kind == JAYESS_VALUE_NULL)) {
        if (headers == NULL) {
            headers = jayess_object_new();
            jayess_object_set_value(env->as.object_value, "headers", jayess_value_from_object(headers));
        }
        if (jayess_http_header_get_ci(headers, "Content-Length") == NULL && !jayess_http_headers_transfer_chunked(headers)) {
            chunk_text = jayess_value_stringify(chunk);
            if (chunk_text != NULL) {
                snprintf(length_text, sizeof(length_text), "%zu", strlen(chunk_text));
                jayess_object_set_value(headers, "Content-Length", jayess_value_from_string(length_text));
            }
            free(chunk_text);
        }
    }
    if (chunk != NULL && !(chunk->kind == JAYESS_VALUE_UNDEFINED || chunk->kind == JAYESS_VALUE_NULL)) {
        if (!jayess_value_as_bool(jayess_std_http_response_write_method(env, chunk))) {
            return env;
        }
    } else if (!state->headers_sent) {
        if (!jayess_std_http_response_send_headers(env)) {
            return env;
        }
    }
    state->finished = 1;
    jayess_object_set_value(env->as.object_value, "finished", jayess_value_from_bool(1));
    if (state->chunked && state->socket != NULL) {
        if (!jayess_value_as_bool(jayess_std_socket_write_method(state->socket, jayess_value_from_string("0\r\n\r\n")))) {
            return env;
        }
    }
    if (state->socket != NULL) {
        if (state->keep_alive) {
            return env;
        }
        jayess_std_socket_close_method(state->socket);
    }
    return env;
}

static jayess_value *jayess_std_http_server_close_method(jayess_value *env) {
    jayess_http_server_state *state;
    if (env == NULL || env->kind != JAYESS_VALUE_OBJECT || env->as.object_value == NULL) {
        return jayess_value_undefined();
    }
    state = (jayess_http_server_state *)env->as.object_value->native_handle;
    if (state == NULL || state->closed) {
        return env;
    }
    state->closed = 1;
    jayess_object_set_value(env->as.object_value, "closed", jayess_value_from_bool(1));
    if (state->backend_server != NULL) {
        jayess_std_server_close_method(state->backend_server);
    }
    return env;
}

static jayess_value *jayess_std_http_server_new(jayess_value *handler, jayess_value *tls_options, int secure, int http_mode, const char *api_name) {
    jayess_object *server;
    jayess_value *server_value;
    jayess_http_server_state *state;
    if (handler == NULL || handler->kind != JAYESS_VALUE_FUNCTION) {
        char message[96];
        snprintf(message, sizeof(message), "%s handler must be a function", api_name != NULL ? api_name : "server.createServer");
        jayess_throw(jayess_type_error_value(message));
        return jayess_value_undefined();
    }
    server = jayess_object_new();
    if (server == NULL) {
        return jayess_value_undefined();
    }
    server_value = jayess_value_from_object(server);
    state = (jayess_http_server_state *)malloc(sizeof(jayess_http_server_state));
    if (state == NULL) {
        return jayess_value_undefined();
    }
    state->handler = handler;
    state->tls_options = tls_options;
    state->backend_server = NULL;
    state->secure = secure;
    state->http_mode = http_mode;
    state->closed = 0;
    server->native_handle = state;
    jayess_object_set_value(server, "listening", jayess_value_from_bool(0));
    jayess_object_set_value(server, "closed", jayess_value_from_bool(0));
    jayess_object_set_value(server, "secure", jayess_value_from_bool(secure));
    jayess_object_set_value(server, "listen", jayess_value_from_function((void *)jayess_std_http_server_listen_method, server_value, "listen", NULL, 2, 0));
    jayess_object_set_value(server, "close", jayess_value_from_function((void *)jayess_std_http_server_close_method, server_value, "close", NULL, 0, 0));
    return server_value;
}

static jayess_value *jayess_std_http_server_listen_method(jayess_value *env, jayess_value *port_value, jayess_value *host_value) {
    jayess_http_server_state *state;
    jayess_object *options;
    char *host_text;
    int port;
    if (env == NULL || env->kind != JAYESS_VALUE_OBJECT || env->as.object_value == NULL) {
        return jayess_value_undefined();
    }
    state = (jayess_http_server_state *)env->as.object_value->native_handle;
    if (state == NULL || state->handler == NULL) {
        return jayess_value_undefined();
    }
    host_text = jayess_value_stringify(host_value);
    port = (int)jayess_value_to_number(port_value);
    options = jayess_object_new();
    jayess_object_set_value(options, "host", jayess_value_from_string(host_text != NULL && host_text[0] != '\0' ? host_text : "127.0.0.1"));
    jayess_object_set_value(options, "port", jayess_value_from_number((double)port));
    free(host_text);
    state->backend_server = jayess_std_net_listen(jayess_value_from_object(options));
    if (state->backend_server == NULL || state->backend_server->kind != JAYESS_VALUE_OBJECT) {
        return jayess_value_undefined();
    }
    jayess_object_set_value(env->as.object_value, "listening", jayess_value_from_bool(1));
    while (!state->closed) {
        jayess_value *socket = jayess_std_server_accept_method(state->backend_server);
        jayess_value *request;
        jayess_object *response_object;
        jayess_http_response_state *response_state;
        jayess_value *response;
        if (state->closed) {
            break;
        }
        if (socket == NULL || socket->kind != JAYESS_VALUE_OBJECT || !jayess_std_kind_is(socket, "Socket")) {
            continue;
        }
        if (state->secure) {
            socket = jayess_std_tls_accept_socket(socket, state->tls_options);
            if (jayess_has_exception()) {
                break;
            }
            if (socket == NULL || socket->kind != JAYESS_VALUE_OBJECT || !jayess_std_kind_is(socket, "Socket")) {
                continue;
            }
        }
        if (!state->http_mode) {
            jayess_value_call_one(state->handler, socket);
            if (jayess_has_exception()) {
                jayess_std_socket_close_method(socket);
                break;
            }
            if (!jayess_value_as_bool(jayess_object_get(socket->as.object_value, "closed"))) {
                jayess_std_socket_close_method(socket);
            }
            continue;
        }
        while (!state->closed) {
            request = jayess_http_read_request_from_socket(socket);
            if (request == NULL || request->kind != JAYESS_VALUE_OBJECT) {
                jayess_std_socket_close_method(socket);
                break;
            }
            jayess_object_set_value(request->as.object_value, "url", jayess_object_get(request->as.object_value, "path"));
            jayess_object_set_value(request->as.object_value, "keepAlive", jayess_value_from_bool(jayess_http_request_wants_keep_alive(request)));
            response_object = jayess_object_new();
            response_state = (jayess_http_response_state *)malloc(sizeof(jayess_http_response_state));
            if (response_state == NULL) {
                jayess_std_socket_close_method(socket);
                break;
            }
            response_state->socket = socket;
            response_state->headers_sent = 0;
            response_state->finished = 0;
            response_state->keep_alive = jayess_http_request_wants_keep_alive(request);
            response_state->chunked = 0;
            response = jayess_value_from_object(response_object);
            response_object->native_handle = response_state;
            jayess_object_set_value(response_object, "statusCode", jayess_value_from_number(200));
            jayess_object_set_value(response_object, "headers", jayess_value_from_object(jayess_object_new()));
            jayess_object_set_value(response_object, "finished", jayess_value_from_bool(0));
            jayess_object_set_value(response_object, "setHeader", jayess_value_from_function((void *)jayess_std_http_response_set_header_method, response, "setHeader", NULL, 2, 0));
            jayess_object_set_value(response_object, "write", jayess_value_from_function((void *)jayess_std_http_response_write_method, response, "write", NULL, 1, 0));
            jayess_object_set_value(response_object, "end", jayess_value_from_function((void *)jayess_std_http_response_end_method, response, "end", NULL, 1, 0));
            jayess_value_call_two_with_this(state->handler, jayess_value_undefined(), request, response);
            if (jayess_has_exception()) {
                jayess_std_socket_close_method(socket);
                free(response_state);
                response_object->native_handle = NULL;
                break;
            }
            if (!response_state->finished) {
                jayess_std_http_response_end_method(response, jayess_value_undefined());
            }
            {
                int keep_socket = response_state->keep_alive && !jayess_value_as_bool(jayess_object_get(socket->as.object_value, "closed"));
                free(response_state);
                response_object->native_handle = NULL;
                if (!keep_socket) {
                    break;
                }
            }
        }
    }
    jayess_object_set_value(env->as.object_value, "listening", jayess_value_from_bool(0));
    return env;
}

jayess_value *jayess_std_http_create_server(jayess_value *handler) {
    return jayess_std_http_server_new(handler, jayess_value_undefined(), 0, 1, "http.createServer");
}

jayess_value *jayess_std_https_create_server(jayess_value *options, jayess_value *handler) {
    if (jayess_value_as_object(options) == NULL) {
        jayess_throw(jayess_type_error_value("https.createServer options must be an object"));
        return jayess_value_undefined();
    }
    return jayess_std_http_server_new(handler, options, 1, 1, "https.createServer");
}

jayess_value *jayess_std_tls_create_server(jayess_value *options, jayess_value *handler) {
    if (jayess_value_as_object(options) == NULL) {
        jayess_throw(jayess_type_error_value("tls.createServer options must be an object"));
        return jayess_value_undefined();
    }
    return jayess_std_http_server_new(handler, options, 1, 0, "tls.createServer");
}

jayess_value *jayess_std_http_request(jayess_value *options) {
    jayess_object *object = jayess_value_as_object(options);
    jayess_object *request_object = NULL;
    if (object == NULL) {
        return jayess_value_undefined();
    }
    if (jayess_object_get(object, "url") != NULL && !jayess_value_is_nullish(jayess_object_get(object, "url"))) {
        request_object = jayess_http_request_object_from_url_value(jayess_object_get(object, "url"), jayess_value_as_string(jayess_object_get(object, "method")) != NULL ? jayess_value_as_string(jayess_object_get(object, "method")) : "GET");
        if (request_object != NULL) {
            if (jayess_object_get(object, "version") != NULL) {
                jayess_object_set_value(request_object, "version", jayess_object_get(object, "version"));
            }
            if (jayess_object_get(object, "headers") != NULL) {
                jayess_object_set_value(request_object, "headers", jayess_object_get(object, "headers"));
            }
            if (jayess_object_get(object, "body") != NULL) {
                jayess_object_set_value(request_object, "body", jayess_object_get(object, "body"));
            }
            if (jayess_object_get(object, "host") != NULL) {
                jayess_object_set_value(request_object, "host", jayess_object_get(object, "host"));
            }
            if (jayess_object_get(object, "port") != NULL) {
                jayess_object_set_value(request_object, "port", jayess_object_get(object, "port"));
            }
            if (jayess_object_get(object, "timeout") != NULL) {
                jayess_object_set_value(request_object, "timeout", jayess_object_get(object, "timeout"));
            }
        }
    } else {
        request_object = jayess_object_new();
        jayess_object_set_value(request_object, "method", jayess_object_get(object, "method") != NULL ? jayess_object_get(object, "method") : jayess_value_from_string("GET"));
        jayess_object_set_value(request_object, "path", jayess_object_get(object, "path") != NULL ? jayess_object_get(object, "path") : jayess_value_from_string("/"));
        jayess_object_set_value(request_object, "version", jayess_object_get(object, "version") != NULL ? jayess_object_get(object, "version") : jayess_value_from_string("HTTP/1.1"));
        jayess_object_set_value(request_object, "headers", jayess_object_get(object, "headers") != NULL ? jayess_object_get(object, "headers") : jayess_value_from_object(jayess_object_new()));
        jayess_object_set_value(request_object, "body", jayess_object_get(object, "body") != NULL ? jayess_object_get(object, "body") : jayess_value_from_string(""));
        jayess_object_set_value(request_object, "host", jayess_object_get(object, "host"));
        jayess_object_set_value(request_object, "port", jayess_object_get(object, "port") != NULL ? jayess_object_get(object, "port") : jayess_value_from_number(80));
        jayess_object_set_value(request_object, "timeout", jayess_object_get(object, "timeout") != NULL ? jayess_object_get(object, "timeout") : jayess_value_from_number(0));
    }
    if (request_object == NULL) {
        return jayess_value_undefined();
    }
    if (jayess_http_text_eq_ci(jayess_value_as_string(jayess_object_get(request_object, "scheme")), "https")) {
        jayess_throw(jayess_type_error_value("https URLs must use https.request(...) or https.get(...)"));
        return jayess_value_undefined();
    }
    if (jayess_value_as_object(jayess_object_get(request_object, "headers")) != NULL) {
        jayess_object *headers = jayess_value_as_object(jayess_object_get(request_object, "headers"));
        if (jayess_object_get(headers, "Host") == NULL && jayess_object_get(request_object, "host") != NULL) {
            jayess_object_set_value(headers, "Host", jayess_object_get(request_object, "host"));
        }
        if (jayess_object_get(headers, "Connection") == NULL) {
            jayess_object_set_value(headers, "Connection", jayess_value_from_string("close"));
        }
    }
    return jayess_http_request_from_parts(request_object);
}

jayess_value *jayess_std_http_get(jayess_value *input) {
    jayess_object *request_object = NULL;
    if (input != NULL && input->kind == JAYESS_VALUE_STRING) {
        request_object = jayess_http_request_object_from_url_value(input, "GET");
    } else {
        jayess_object *input_object = jayess_value_as_object(input);
        if (input_object == NULL) {
            return jayess_value_undefined();
        }
        if (jayess_object_get(input_object, "url") != NULL && !jayess_value_is_nullish(jayess_object_get(input_object, "url"))) {
            request_object = jayess_http_request_object_from_url_value(jayess_object_get(input_object, "url"), "GET");
            if (request_object != NULL) {
                if (jayess_object_get(input_object, "version") != NULL) {
                    jayess_object_set_value(request_object, "version", jayess_object_get(input_object, "version"));
                }
                if (jayess_object_get(input_object, "headers") != NULL) {
                    jayess_object_set_value(request_object, "headers", jayess_object_get(input_object, "headers"));
                }
                if (jayess_object_get(input_object, "host") != NULL) {
                    jayess_object_set_value(request_object, "host", jayess_object_get(input_object, "host"));
                }
                if (jayess_object_get(input_object, "port") != NULL) {
                    jayess_object_set_value(request_object, "port", jayess_object_get(input_object, "port"));
                }
                if (jayess_object_get(input_object, "timeout") != NULL) {
                    jayess_object_set_value(request_object, "timeout", jayess_object_get(input_object, "timeout"));
                }
            }
        } else {
            request_object = jayess_object_new();
            jayess_object_set_value(request_object, "method", jayess_value_from_string("GET"));
            jayess_object_set_value(request_object, "path", jayess_object_get(input_object, "path") != NULL ? jayess_object_get(input_object, "path") : jayess_value_from_string("/"));
            jayess_object_set_value(request_object, "version", jayess_object_get(input_object, "version") != NULL ? jayess_object_get(input_object, "version") : jayess_value_from_string("HTTP/1.1"));
            jayess_object_set_value(request_object, "headers", jayess_object_get(input_object, "headers") != NULL ? jayess_object_get(input_object, "headers") : jayess_value_from_object(jayess_object_new()));
            jayess_object_set_value(request_object, "body", jayess_value_from_string(""));
            jayess_object_set_value(request_object, "host", jayess_object_get(input_object, "host"));
            jayess_object_set_value(request_object, "port", jayess_object_get(input_object, "port") != NULL ? jayess_object_get(input_object, "port") : jayess_value_from_number(80));
            jayess_object_set_value(request_object, "timeout", jayess_object_get(input_object, "timeout") != NULL ? jayess_object_get(input_object, "timeout") : jayess_value_from_number(0));
        }
    }
    if (request_object == NULL) {
        return jayess_value_undefined();
    }
    if (jayess_http_text_eq_ci(jayess_value_as_string(jayess_object_get(request_object, "scheme")), "https")) {
        jayess_throw(jayess_type_error_value("https URLs must use https.request(...) or https.get(...)"));
        return jayess_value_undefined();
    }
    if (jayess_value_as_object(jayess_object_get(request_object, "headers")) != NULL) {
        jayess_object *headers = jayess_value_as_object(jayess_object_get(request_object, "headers"));
        if (jayess_object_get(headers, "Host") == NULL && jayess_object_get(request_object, "host") != NULL) {
            jayess_object_set_value(headers, "Host", jayess_object_get(request_object, "host"));
        }
        if (jayess_object_get(headers, "Connection") == NULL) {
            jayess_object_set_value(headers, "Connection", jayess_value_from_string("close"));
        }
    }
    return jayess_http_request_from_parts(request_object);
}

jayess_value *jayess_std_http_request_stream(jayess_value *options) {
    jayess_object *object = jayess_value_as_object(options);
    jayess_object *request_object = NULL;
    if (object == NULL) {
        return jayess_value_undefined();
    }
    if (jayess_object_get(object, "url") != NULL && !jayess_value_is_nullish(jayess_object_get(object, "url"))) {
        request_object = jayess_http_request_object_from_url_value(jayess_object_get(object, "url"), jayess_value_as_string(jayess_object_get(object, "method")) != NULL ? jayess_value_as_string(jayess_object_get(object, "method")) : "GET");
        if (request_object != NULL) {
            if (jayess_object_get(object, "version") != NULL) {
                jayess_object_set_value(request_object, "version", jayess_object_get(object, "version"));
            }
            if (jayess_object_get(object, "headers") != NULL) {
                jayess_object_set_value(request_object, "headers", jayess_object_get(object, "headers"));
            }
            if (jayess_object_get(object, "body") != NULL) {
                jayess_object_set_value(request_object, "body", jayess_object_get(object, "body"));
            }
            if (jayess_object_get(object, "host") != NULL) {
                jayess_object_set_value(request_object, "host", jayess_object_get(object, "host"));
            }
            if (jayess_object_get(object, "port") != NULL) {
                jayess_object_set_value(request_object, "port", jayess_object_get(object, "port"));
            }
            if (jayess_object_get(object, "timeout") != NULL) {
                jayess_object_set_value(request_object, "timeout", jayess_object_get(object, "timeout"));
            }
        }
    } else {
        request_object = jayess_object_new();
        jayess_object_set_value(request_object, "method", jayess_object_get(object, "method") != NULL ? jayess_object_get(object, "method") : jayess_value_from_string("GET"));
        jayess_object_set_value(request_object, "path", jayess_object_get(object, "path") != NULL ? jayess_object_get(object, "path") : jayess_value_from_string("/"));
        jayess_object_set_value(request_object, "version", jayess_object_get(object, "version") != NULL ? jayess_object_get(object, "version") : jayess_value_from_string("HTTP/1.1"));
        jayess_object_set_value(request_object, "headers", jayess_object_get(object, "headers") != NULL ? jayess_object_get(object, "headers") : jayess_value_from_object(jayess_object_new()));
        jayess_object_set_value(request_object, "body", jayess_object_get(object, "body") != NULL ? jayess_object_get(object, "body") : jayess_value_from_string(""));
        jayess_object_set_value(request_object, "host", jayess_object_get(object, "host"));
        jayess_object_set_value(request_object, "port", jayess_object_get(object, "port") != NULL ? jayess_object_get(object, "port") : jayess_value_from_number(80));
        jayess_object_set_value(request_object, "timeout", jayess_object_get(object, "timeout") != NULL ? jayess_object_get(object, "timeout") : jayess_value_from_number(0));
    }
    if (request_object == NULL) {
        return jayess_value_undefined();
    }
    if (jayess_http_text_eq_ci(jayess_value_as_string(jayess_object_get(request_object, "scheme")), "https")) {
        jayess_throw(jayess_type_error_value("https URLs must use https.request(...) or https.get(...)"));
        return jayess_value_undefined();
    }
    if (jayess_value_as_object(jayess_object_get(request_object, "headers")) != NULL) {
        jayess_object *headers = jayess_value_as_object(jayess_object_get(request_object, "headers"));
        if (jayess_object_get(headers, "Host") == NULL && jayess_object_get(request_object, "host") != NULL) {
            jayess_object_set_value(headers, "Host", jayess_object_get(request_object, "host"));
        }
        if (jayess_object_get(headers, "Connection") == NULL) {
            jayess_object_set_value(headers, "Connection", jayess_value_from_string("close"));
        }
    }
    return jayess_http_request_stream_from_parts(request_object);
}

jayess_value *jayess_std_http_get_stream(jayess_value *input) {
    jayess_object *request_object = NULL;
    if (input != NULL && input->kind == JAYESS_VALUE_STRING) {
        request_object = jayess_http_request_object_from_url_value(input, "GET");
    } else {
        jayess_object *input_object = jayess_value_as_object(input);
        if (input_object == NULL) {
            return jayess_value_undefined();
        }
        if (jayess_object_get(input_object, "url") != NULL && !jayess_value_is_nullish(jayess_object_get(input_object, "url"))) {
            request_object = jayess_http_request_object_from_url_value(jayess_object_get(input_object, "url"), "GET");
            if (request_object != NULL) {
                if (jayess_object_get(input_object, "version") != NULL) {
                    jayess_object_set_value(request_object, "version", jayess_object_get(input_object, "version"));
                }
                if (jayess_object_get(input_object, "headers") != NULL) {
                    jayess_object_set_value(request_object, "headers", jayess_object_get(input_object, "headers"));
                }
                if (jayess_object_get(input_object, "host") != NULL) {
                    jayess_object_set_value(request_object, "host", jayess_object_get(input_object, "host"));
                }
                if (jayess_object_get(input_object, "port") != NULL) {
                    jayess_object_set_value(request_object, "port", jayess_object_get(input_object, "port"));
                }
                if (jayess_object_get(input_object, "timeout") != NULL) {
                    jayess_object_set_value(request_object, "timeout", jayess_object_get(input_object, "timeout"));
                }
            }
        } else {
            request_object = jayess_object_new();
            jayess_object_set_value(request_object, "method", jayess_value_from_string("GET"));
            jayess_object_set_value(request_object, "path", jayess_object_get(input_object, "path") != NULL ? jayess_object_get(input_object, "path") : jayess_value_from_string("/"));
            jayess_object_set_value(request_object, "version", jayess_object_get(input_object, "version") != NULL ? jayess_object_get(input_object, "version") : jayess_value_from_string("HTTP/1.1"));
            jayess_object_set_value(request_object, "headers", jayess_object_get(input_object, "headers") != NULL ? jayess_object_get(input_object, "headers") : jayess_value_from_object(jayess_object_new()));
            jayess_object_set_value(request_object, "body", jayess_value_from_string(""));
            jayess_object_set_value(request_object, "host", jayess_object_get(input_object, "host"));
            jayess_object_set_value(request_object, "port", jayess_object_get(input_object, "port") != NULL ? jayess_object_get(input_object, "port") : jayess_value_from_number(80));
            jayess_object_set_value(request_object, "timeout", jayess_object_get(input_object, "timeout") != NULL ? jayess_object_get(input_object, "timeout") : jayess_value_from_number(0));
        }
    }
    if (request_object == NULL) {
        return jayess_value_undefined();
    }
    if (jayess_http_text_eq_ci(jayess_value_as_string(jayess_object_get(request_object, "scheme")), "https")) {
        jayess_throw(jayess_type_error_value("https URLs must use https.request(...) or https.get(...)"));
        return jayess_value_undefined();
    }
    if (jayess_value_as_object(jayess_object_get(request_object, "headers")) != NULL) {
        jayess_object *headers = jayess_value_as_object(jayess_object_get(request_object, "headers"));
        if (jayess_object_get(headers, "Host") == NULL && jayess_object_get(request_object, "host") != NULL) {
            jayess_object_set_value(headers, "Host", jayess_object_get(request_object, "host"));
        }
        if (jayess_object_get(headers, "Connection") == NULL) {
            jayess_object_set_value(headers, "Connection", jayess_value_from_string("close"));
        }
    }
    return jayess_http_request_stream_from_parts(request_object);
}

jayess_value *jayess_std_http_request_async(jayess_value *options) {
    jayess_value *promise = jayess_std_promise_pending();
    jayess_enqueue_http_request_task(promise, options, 0, 0, 0);
    return promise;
}

jayess_value *jayess_std_http_get_async(jayess_value *input) {
    jayess_value *promise = jayess_std_promise_pending();
    jayess_enqueue_http_request_task(promise, input, 1, 0, 0);
    return promise;
}

jayess_value *jayess_std_http_request_stream_async(jayess_value *options) {
    jayess_value *promise = jayess_std_promise_pending();
    jayess_enqueue_http_request_task(promise, options, 0, 0, 1);
    return promise;
}

jayess_value *jayess_std_http_get_stream_async(jayess_value *input) {
    jayess_value *promise = jayess_std_promise_pending();
    jayess_enqueue_http_request_task(promise, input, 1, 0, 1);
    return promise;
}

static unsigned char *jayess_http_request_body_bytes(jayess_value *body_value, size_t *length_out) {
    unsigned char *buffer = NULL;
    size_t length = 0;
    if (length_out != NULL) {
        *length_out = 0;
    }
    if (body_value == NULL || jayess_value_is_nullish(body_value)) {
        return NULL;
    }
    if (body_value != NULL && body_value->kind == JAYESS_VALUE_OBJECT && jayess_std_kind_is(body_value, "Uint8Array")) {
        jayess_array *bytes = jayess_std_bytes_slot(body_value);
        int i;
        if (bytes == NULL || bytes->count <= 0) {
            return NULL;
        }
        length = (size_t)bytes->count;
        buffer = (unsigned char *)malloc(length);
        if (buffer == NULL) {
            return NULL;
        }
        for (i = 0; i < bytes->count; i++) {
            buffer[i] = (unsigned char)((int)jayess_value_to_number(jayess_array_get(bytes, i)) & 255);
        }
        if (length_out != NULL) {
            *length_out = length;
        }
        return buffer;
    }
    {
        char *text = jayess_value_stringify(body_value);
        if (text == NULL) {
            return NULL;
        }
        length = strlen(text);
        if (length == 0) {
            free(text);
            return NULL;
        }
        buffer = (unsigned char *)malloc(length);
        if (buffer == NULL) {
            free(text);
            return NULL;
        }
        memcpy(buffer, text, length);
        free(text);
        if (length_out != NULL) {
            *length_out = length;
        }
        return buffer;
    }
}

#ifdef _WIN32
static jayess_value *jayess_http_body_stream_new_winhttp(HINTERNET request, HINTERNET connection, HINTERNET session, jayess_object *headers) {
    jayess_object *object;
    jayess_winhttp_stream_state *state;
    long content_length;
    if (request == NULL || connection == NULL || session == NULL) {
        if (request != NULL) {
            WinHttpCloseHandle(request);
        }
        if (connection != NULL) {
            WinHttpCloseHandle(connection);
        }
        if (session != NULL) {
            WinHttpCloseHandle(session);
        }
        return jayess_value_undefined();
    }
    object = jayess_object_new();
    if (object == NULL) {
        WinHttpCloseHandle(request);
        WinHttpCloseHandle(connection);
        WinHttpCloseHandle(session);
        return jayess_value_undefined();
    }
    state = (jayess_winhttp_stream_state *)malloc(sizeof(jayess_winhttp_stream_state));
    if (state == NULL) {
        WinHttpCloseHandle(request);
        WinHttpCloseHandle(connection);
        WinHttpCloseHandle(session);
        return jayess_value_undefined();
    }
    state->request = request;
    state->connection = connection;
    state->session = session;
    object->native_handle = state;
    content_length = jayess_http_headers_content_length(headers);
    jayess_object_set_value(object, "__jayess_std_kind", jayess_value_from_string("HttpBodyStream"));
    jayess_object_set_value(object, "closed", jayess_value_from_bool(0));
    jayess_object_set_value(object, "readableEnded", jayess_value_from_bool(0));
    jayess_object_set_value(object, "errored", jayess_value_from_bool(0));
    jayess_object_set_value(object, "__jayess_http_body_chunked", jayess_value_from_bool(0));
    jayess_object_set_value(object, "__jayess_http_body_remaining", jayess_value_from_number((double)content_length));
    jayess_object_set_value(object, "__jayess_http_body_chunk_remaining", jayess_value_from_number(-1));
    jayess_object_set_value(object, "__jayess_http_body_chunk_finished", jayess_value_from_bool(0));
    jayess_object_set_value(object, "__jayess_http_body_prebuffer", jayess_std_uint8_array_from_bytes((const unsigned char *)"", 0));
    jayess_object_set_value(object, "__jayess_http_body_prebuffer_offset", jayess_value_from_number(0));
    if (content_length == 0) {
        jayess_value *stream_value = jayess_value_from_object(object);
        jayess_http_body_stream_mark_ended(stream_value);
        jayess_http_body_stream_close_native(stream_value);
        return stream_value;
    }
    return jayess_value_from_object(object);
}

static jayess_value *jayess_https_read_response_stream(HINTERNET request, HINTERNET connection, HINTERNET session) {
    DWORD header_bytes = 0;
    wchar_t *raw_headers_w = NULL;
    char *raw_headers = NULL;
    char *version = NULL;
    char *status_text = NULL;
    char *reason = NULL;
    char *header_lines = NULL;
    jayess_object *headers = NULL;
    jayess_object *result = NULL;
    DWORD status_code = 0;
    DWORD status_size = sizeof(status_code);

    WinHttpQueryHeaders(request, WINHTTP_QUERY_RAW_HEADERS_CRLF, WINHTTP_HEADER_NAME_BY_INDEX, WINHTTP_NO_OUTPUT_BUFFER, &header_bytes, WINHTTP_NO_HEADER_INDEX);
    if (GetLastError() != ERROR_INSUFFICIENT_BUFFER || header_bytes == 0) {
        goto cleanup;
    }
    raw_headers_w = (wchar_t *)malloc((size_t)header_bytes);
    if (raw_headers_w == NULL) {
        goto cleanup;
    }
    if (!WinHttpQueryHeaders(request, WINHTTP_QUERY_RAW_HEADERS_CRLF, WINHTTP_HEADER_NAME_BY_INDEX, raw_headers_w, &header_bytes, WINHTTP_NO_HEADER_INDEX)) {
        goto cleanup;
    }
    raw_headers = jayess_wide_to_utf8(raw_headers_w);
    if (raw_headers == NULL) {
        goto cleanup;
    }
    {
        const char *line_end = jayess_http_line_end(raw_headers);
        const char *sp1 = raw_headers;
        const char *sp2;
        const char *header_start;
        while (sp1 < line_end && *sp1 != ' ') {
            sp1++;
        }
        if (sp1 >= line_end) {
            goto cleanup;
        }
        sp2 = sp1 + 1;
        while (sp2 < line_end && *sp2 != ' ') {
            sp2++;
        }
        version = jayess_substring(raw_headers, 0, (size_t)(sp1 - raw_headers));
        status_text = jayess_substring(sp1 + 1, 0, (size_t)((sp2 < line_end ? sp2 : line_end) - sp1 - 1));
        reason = sp2 < line_end ? jayess_substring(sp2 + 1, 0, (size_t)(line_end - sp2 - 1)) : jayess_strdup("");
        header_start = jayess_http_next_line(line_end);
        header_lines = jayess_substring(header_start, 0, strlen(header_start));
    }
    headers = jayess_http_parse_header_object(header_lines != NULL ? header_lines : "");
    if (!WinHttpQueryHeaders(request, WINHTTP_QUERY_STATUS_CODE | WINHTTP_QUERY_FLAG_NUMBER, WINHTTP_HEADER_NAME_BY_INDEX, &status_code, &status_size, WINHTTP_NO_HEADER_INDEX)) {
        status_code = (DWORD)atoi(status_text != NULL ? status_text : "0");
    }
    result = jayess_object_new();
    if (result == NULL) {
        goto cleanup;
    }
    jayess_object_set_value(result, "version", jayess_value_from_string(version != NULL ? version : "HTTP/1.1"));
    jayess_object_set_value(result, "status", jayess_value_from_number((double)status_code));
    jayess_object_set_value(result, "reason", jayess_value_from_string(reason != NULL ? reason : ""));
    jayess_object_set_value(result, "statusText", jayess_value_from_string(reason != NULL ? reason : ""));
    jayess_object_set_value(result, "ok", jayess_value_from_bool(status_code >= 200 && status_code < 300));
    jayess_object_set_value(result, "headers", jayess_value_from_object(headers != NULL ? headers : jayess_object_new()));
    jayess_object_set_value(result, "bodyStream", jayess_http_body_stream_new_winhttp(request, connection, session, headers));
    request = NULL;
    connection = NULL;
    session = NULL;
    free(raw_headers_w);
    free(raw_headers);
    free(version);
    free(status_text);
    free(reason);
    free(header_lines);
    return jayess_value_from_object(result);

cleanup:
    if (request != NULL) {
        WinHttpCloseHandle(request);
    }
    if (connection != NULL) {
        WinHttpCloseHandle(connection);
    }
    if (session != NULL) {
        WinHttpCloseHandle(session);
    }
    free(raw_headers_w);
    free(raw_headers);
    free(version);
    free(status_text);
    free(reason);
    free(header_lines);
    return jayess_value_undefined();
}

static jayess_value *jayess_https_request_stream_from_parts(jayess_object *request_object) {
    return jayess_https_request_via_tls_from_parts(request_object, 1);
}

static jayess_value *jayess_https_request_from_parts(jayess_object *request_object) {
    return jayess_https_request_via_tls_from_parts(request_object, 0);
}
#else
static jayess_value *jayess_https_request_from_parts(jayess_object *request_object) {
    return jayess_https_request_via_tls_from_parts(request_object, 0);
}
#endif

#ifndef _WIN32
static jayess_value *jayess_https_request_stream_from_parts(jayess_object *request_object) {
    return jayess_https_request_via_tls_from_parts(request_object, 1);
}
#endif

jayess_value *jayess_std_https_request(jayess_value *options) {
    jayess_object *object = jayess_value_as_object(options);
    jayess_object *request_object = NULL;
    if (object == NULL) {
        return jayess_value_undefined();
    }
    {
        const char *method = jayess_value_as_string(jayess_object_get(object, "method"));
        size_t body_length = 0;
        unsigned char *body_bytes = jayess_http_request_body_bytes(jayess_object_get(object, "body"), &body_length);
        free(body_bytes);
        if ((method == NULL || method[0] == '\0' || jayess_http_text_eq_ci(method, "GET")) && body_length == 0) {
            return jayess_std_https_get(options);
        }
    }
    if (jayess_object_get(object, "url") != NULL && !jayess_value_is_nullish(jayess_object_get(object, "url"))) {
        request_object = jayess_http_request_object_from_url_value(jayess_object_get(object, "url"), jayess_value_as_string(jayess_object_get(object, "method")) != NULL ? jayess_value_as_string(jayess_object_get(object, "method")) : "GET");
        if (request_object != NULL) {
            if (jayess_object_get(object, "version") != NULL) {
                jayess_object_set_value(request_object, "version", jayess_object_get(object, "version"));
            }
            if (jayess_object_get(object, "headers") != NULL) {
                jayess_object_set_value(request_object, "headers", jayess_object_get(object, "headers"));
            }
            if (jayess_object_get(object, "body") != NULL) {
                jayess_object_set_value(request_object, "body", jayess_object_get(object, "body"));
            }
            if (jayess_object_get(object, "host") != NULL) {
                jayess_object_set_value(request_object, "host", jayess_object_get(object, "host"));
            }
            if (jayess_object_get(object, "port") != NULL) {
                jayess_object_set_value(request_object, "port", jayess_object_get(object, "port"));
            }
            if (jayess_object_get(object, "timeout") != NULL) {
                jayess_object_set_value(request_object, "timeout", jayess_object_get(object, "timeout"));
            }
            if (jayess_object_get(object, "maxRedirects") != NULL) {
                jayess_object_set_value(request_object, "maxRedirects", jayess_object_get(object, "maxRedirects"));
            }
            jayess_std_https_copy_tls_request_settings(request_object, object);
        }
    } else {
        request_object = jayess_object_new();
        jayess_object_set_value(request_object, "method", jayess_object_get(object, "method") != NULL ? jayess_object_get(object, "method") : jayess_value_from_string("GET"));
        jayess_object_set_value(request_object, "path", jayess_object_get(object, "path") != NULL ? jayess_object_get(object, "path") : jayess_value_from_string("/"));
        jayess_object_set_value(request_object, "scheme", jayess_value_from_string("https"));
        jayess_object_set_value(request_object, "version", jayess_object_get(object, "version") != NULL ? jayess_object_get(object, "version") : jayess_value_from_string("HTTP/1.1"));
        jayess_object_set_value(request_object, "headers", jayess_object_get(object, "headers") != NULL ? jayess_object_get(object, "headers") : jayess_value_from_object(jayess_object_new()));
        jayess_object_set_value(request_object, "body", jayess_object_get(object, "body") != NULL ? jayess_object_get(object, "body") : jayess_value_from_string(""));
        jayess_object_set_value(request_object, "host", jayess_object_get(object, "host"));
        jayess_object_set_value(request_object, "port", jayess_object_get(object, "port") != NULL ? jayess_object_get(object, "port") : jayess_value_from_number(443));
        jayess_object_set_value(request_object, "timeout", jayess_object_get(object, "timeout") != NULL ? jayess_object_get(object, "timeout") : jayess_value_from_number(0));
        jayess_object_set_value(request_object, "maxRedirects", jayess_object_get(object, "maxRedirects") != NULL ? jayess_object_get(object, "maxRedirects") : jayess_value_from_number(5));
        jayess_object_set_value(request_object, "rejectUnauthorized", jayess_object_get(object, "rejectUnauthorized") != NULL ? jayess_object_get(object, "rejectUnauthorized") : jayess_value_from_bool(1));
        jayess_std_https_copy_tls_request_settings(request_object, object);
    }
    if (request_object == NULL) {
        return jayess_value_undefined();
    }
    jayess_object_set_value(request_object, "scheme", jayess_value_from_string("https"));
    if (jayess_value_as_object(jayess_object_get(request_object, "headers")) != NULL) {
        jayess_object *headers = jayess_value_as_object(jayess_object_get(request_object, "headers"));
        if (jayess_http_header_get_ci(headers, "Host") == NULL && jayess_object_get(request_object, "host") != NULL) {
            jayess_object_set_value(headers, "Host", jayess_object_get(request_object, "host"));
        }
        if (jayess_http_header_get_ci(headers, "Connection") == NULL) {
            jayess_object_set_value(headers, "Connection", jayess_value_from_string("close"));
        }
        if (jayess_http_header_get_ci(headers, "Content-Length") == NULL && !jayess_http_headers_transfer_chunked(headers)) {
            size_t body_len = 0;
            unsigned char *body_bytes = jayess_http_request_body_bytes(jayess_object_get(request_object, "body"), &body_len);
            if (body_bytes != NULL || body_len > 0) {
                char body_len_text[32];
                snprintf(body_len_text, sizeof(body_len_text), "%u", (unsigned int)body_len);
                jayess_object_set_value(headers, "Content-Length", jayess_value_from_string(body_len_text));
            }
            free(body_bytes);
        }
    }
    return jayess_https_request_from_parts(request_object);
}

jayess_value *jayess_std_https_request_stream(jayess_value *options) {
    jayess_object *object = jayess_value_as_object(options);
    jayess_object *request_object = NULL;
    if (object == NULL) {
        return jayess_value_undefined();
    }
    if (jayess_object_get(object, "url") != NULL && !jayess_value_is_nullish(jayess_object_get(object, "url"))) {
        request_object = jayess_http_request_object_from_url_value(jayess_object_get(object, "url"), jayess_value_as_string(jayess_object_get(object, "method")) != NULL ? jayess_value_as_string(jayess_object_get(object, "method")) : "GET");
        if (request_object != NULL) {
            if (jayess_object_get(object, "version") != NULL) {
                jayess_object_set_value(request_object, "version", jayess_object_get(object, "version"));
            }
            if (jayess_object_get(object, "headers") != NULL) {
                jayess_object_set_value(request_object, "headers", jayess_object_get(object, "headers"));
            }
            if (jayess_object_get(object, "body") != NULL) {
                jayess_object_set_value(request_object, "body", jayess_object_get(object, "body"));
            }
            if (jayess_object_get(object, "host") != NULL) {
                jayess_object_set_value(request_object, "host", jayess_object_get(object, "host"));
            }
            if (jayess_object_get(object, "port") != NULL) {
                jayess_object_set_value(request_object, "port", jayess_object_get(object, "port"));
            }
            if (jayess_object_get(object, "timeout") != NULL) {
                jayess_object_set_value(request_object, "timeout", jayess_object_get(object, "timeout"));
            }
            if (jayess_object_get(object, "maxRedirects") != NULL) {
                jayess_object_set_value(request_object, "maxRedirects", jayess_object_get(object, "maxRedirects"));
            }
            jayess_std_https_copy_tls_request_settings(request_object, object);
        }
    } else {
        request_object = jayess_object_new();
        jayess_object_set_value(request_object, "method", jayess_object_get(object, "method") != NULL ? jayess_object_get(object, "method") : jayess_value_from_string("GET"));
        jayess_object_set_value(request_object, "path", jayess_object_get(object, "path") != NULL ? jayess_object_get(object, "path") : jayess_value_from_string("/"));
        jayess_object_set_value(request_object, "scheme", jayess_value_from_string("https"));
        jayess_object_set_value(request_object, "version", jayess_object_get(object, "version") != NULL ? jayess_object_get(object, "version") : jayess_value_from_string("HTTP/1.1"));
        jayess_object_set_value(request_object, "headers", jayess_object_get(object, "headers") != NULL ? jayess_object_get(object, "headers") : jayess_value_from_object(jayess_object_new()));
        jayess_object_set_value(request_object, "body", jayess_object_get(object, "body") != NULL ? jayess_object_get(object, "body") : jayess_value_from_string(""));
        jayess_object_set_value(request_object, "host", jayess_object_get(object, "host"));
        jayess_object_set_value(request_object, "port", jayess_object_get(object, "port") != NULL ? jayess_object_get(object, "port") : jayess_value_from_number(443));
        jayess_object_set_value(request_object, "timeout", jayess_object_get(object, "timeout") != NULL ? jayess_object_get(object, "timeout") : jayess_value_from_number(0));
        jayess_object_set_value(request_object, "maxRedirects", jayess_object_get(object, "maxRedirects") != NULL ? jayess_object_get(object, "maxRedirects") : jayess_value_from_number(5));
        jayess_object_set_value(request_object, "rejectUnauthorized", jayess_object_get(object, "rejectUnauthorized") != NULL ? jayess_object_get(object, "rejectUnauthorized") : jayess_value_from_bool(1));
        jayess_std_https_copy_tls_request_settings(request_object, object);
    }
    if (request_object == NULL) {
        return jayess_value_undefined();
    }
    jayess_object_set_value(request_object, "scheme", jayess_value_from_string("https"));
    if (jayess_value_as_object(jayess_object_get(request_object, "headers")) != NULL) {
        jayess_object *headers = jayess_value_as_object(jayess_object_get(request_object, "headers"));
        if (jayess_http_header_get_ci(headers, "Host") == NULL && jayess_object_get(request_object, "host") != NULL) {
            jayess_object_set_value(headers, "Host", jayess_object_get(request_object, "host"));
        }
        if (jayess_http_header_get_ci(headers, "Connection") == NULL) {
            jayess_object_set_value(headers, "Connection", jayess_value_from_string("close"));
        }
        if (jayess_http_header_get_ci(headers, "Content-Length") == NULL && !jayess_http_headers_transfer_chunked(headers)) {
            size_t body_len = 0;
            unsigned char *body_bytes = jayess_http_request_body_bytes(jayess_object_get(request_object, "body"), &body_len);
            if (body_bytes != NULL || body_len > 0) {
                char body_len_text[32];
                snprintf(body_len_text, sizeof(body_len_text), "%u", (unsigned int)body_len);
                jayess_object_set_value(headers, "Content-Length", jayess_value_from_string(body_len_text));
            }
            free(body_bytes);
        }
    }
    return jayess_https_request_stream_from_parts(request_object);
}

static int jayess_https_get_uses_body_or_custom_method(jayess_object *object) {
    const char *method;
    size_t body_length = 0;
    unsigned char *body_bytes;
    if (object == NULL) {
        return 0;
    }
    method = jayess_value_as_string(jayess_object_get(object, "method"));
    body_bytes = jayess_http_request_body_bytes(jayess_object_get(object, "body"), &body_length);
    free(body_bytes);
    return (method != NULL && method[0] != '\0' && !jayess_http_text_eq_ci(method, "GET")) || body_length > 0;
}

jayess_value *jayess_std_https_get(jayess_value *input) {
    jayess_object *request_object = NULL;
    if (input != NULL && input->kind == JAYESS_VALUE_STRING) {
        request_object = jayess_object_new();
        jayess_object_set_value(request_object, "method", jayess_value_from_string("GET"));
        jayess_object_set_value(request_object, "url", input);
    }
    if (request_object == NULL) {
        jayess_object *object = jayess_value_as_object(input);
        if (object == NULL) {
            return jayess_value_undefined();
        }
        if (jayess_https_get_uses_body_or_custom_method(object)) {
            jayess_throw(jayess_type_error_value("HTTPS request bodies and custom methods are not supported yet"));
            return jayess_value_undefined();
        }
        request_object = jayess_object_new();
        jayess_object_set_value(request_object, "method", jayess_value_from_string("GET"));
        if (jayess_object_get(object, "url") != NULL && !jayess_value_is_nullish(jayess_object_get(object, "url"))) {
            request_object = jayess_http_request_object_from_url_value(jayess_object_get(object, "url"), "GET");
        } else {
            jayess_object_set_value(request_object, "path", jayess_object_get(object, "path") != NULL ? jayess_object_get(object, "path") : jayess_value_from_string("/"));
            jayess_object_set_value(request_object, "host", jayess_object_get(object, "host"));
            jayess_object_set_value(request_object, "port", jayess_object_get(object, "port") != NULL ? jayess_object_get(object, "port") : jayess_value_from_number(443));
            jayess_object_set_value(request_object, "scheme", jayess_value_from_string("https"));
            jayess_object_set_value(request_object, "version", jayess_object_get(object, "version") != NULL ? jayess_object_get(object, "version") : jayess_value_from_string("HTTP/1.1"));
            jayess_object_set_value(request_object, "headers", jayess_object_get(object, "headers") != NULL ? jayess_object_get(object, "headers") : jayess_value_from_object(jayess_object_new()));
            jayess_object_set_value(request_object, "body", jayess_value_from_string(""));
            jayess_object_set_value(request_object, "timeout", jayess_object_get(object, "timeout") != NULL ? jayess_object_get(object, "timeout") : jayess_value_from_number(0));
            jayess_object_set_value(request_object, "maxRedirects", jayess_object_get(object, "maxRedirects") != NULL ? jayess_object_get(object, "maxRedirects") : jayess_value_from_number(5));
            jayess_object_set_value(request_object, "rejectUnauthorized", jayess_object_get(object, "rejectUnauthorized") != NULL ? jayess_object_get(object, "rejectUnauthorized") : jayess_value_from_bool(1));
            jayess_std_https_copy_tls_request_settings(request_object, object);
        }
        if (request_object != NULL && jayess_object_get(object, "headers") != NULL) {
            jayess_object_set_value(request_object, "headers", jayess_object_get(object, "headers"));
        }
        if (request_object != NULL && jayess_object_get(object, "version") != NULL) {
            jayess_object_set_value(request_object, "version", jayess_object_get(object, "version"));
        }
        if (request_object != NULL && jayess_object_get(object, "host") != NULL) {
            jayess_object_set_value(request_object, "host", jayess_object_get(object, "host"));
        }
        if (request_object != NULL && jayess_object_get(object, "port") != NULL) {
            jayess_object_set_value(request_object, "port", jayess_object_get(object, "port"));
        }
        if (request_object != NULL && jayess_object_get(object, "timeout") != NULL) {
            jayess_object_set_value(request_object, "timeout", jayess_object_get(object, "timeout"));
        }
        if (request_object != NULL && jayess_object_get(object, "maxRedirects") != NULL) {
            jayess_object_set_value(request_object, "maxRedirects", jayess_object_get(object, "maxRedirects"));
        }
        if (request_object != NULL) {
            jayess_std_https_copy_tls_request_settings(request_object, object);
        }
    }
    if (request_object == NULL) {
        return jayess_value_undefined();
    }
    jayess_object_set_value(request_object, "scheme", jayess_value_from_string("https"));
    if (jayess_value_as_object(jayess_object_get(request_object, "headers")) != NULL) {
        jayess_object *headers = jayess_value_as_object(jayess_object_get(request_object, "headers"));
        if (jayess_http_header_get_ci(headers, "Host") == NULL && jayess_object_get(request_object, "host") != NULL) {
            jayess_object_set_value(headers, "Host", jayess_object_get(request_object, "host"));
        }
    }
    return jayess_https_request_from_parts(request_object);
}

jayess_value *jayess_std_https_get_stream(jayess_value *input) {
    jayess_object *request_object = NULL;
    if (input != NULL && input->kind == JAYESS_VALUE_STRING) {
        request_object = jayess_object_new();
        jayess_object_set_value(request_object, "method", jayess_value_from_string("GET"));
        jayess_object_set_value(request_object, "url", input);
    }
    if (request_object == NULL) {
        jayess_object *object = jayess_value_as_object(input);
        if (object == NULL) {
            return jayess_value_undefined();
        }
        if (jayess_https_get_uses_body_or_custom_method(object)) {
            jayess_throw(jayess_type_error_value("HTTPS request bodies and custom methods are not supported yet"));
            return jayess_value_undefined();
        }
        request_object = jayess_object_new();
        jayess_object_set_value(request_object, "method", jayess_value_from_string("GET"));
        if (jayess_object_get(object, "url") != NULL && !jayess_value_is_nullish(jayess_object_get(object, "url"))) {
            request_object = jayess_http_request_object_from_url_value(jayess_object_get(object, "url"), "GET");
        } else {
            jayess_object_set_value(request_object, "path", jayess_object_get(object, "path") != NULL ? jayess_object_get(object, "path") : jayess_value_from_string("/"));
            jayess_object_set_value(request_object, "host", jayess_object_get(object, "host"));
            jayess_object_set_value(request_object, "port", jayess_object_get(object, "port") != NULL ? jayess_object_get(object, "port") : jayess_value_from_number(443));
            jayess_object_set_value(request_object, "scheme", jayess_value_from_string("https"));
            jayess_object_set_value(request_object, "version", jayess_object_get(object, "version") != NULL ? jayess_object_get(object, "version") : jayess_value_from_string("HTTP/1.1"));
            jayess_object_set_value(request_object, "headers", jayess_object_get(object, "headers") != NULL ? jayess_object_get(object, "headers") : jayess_value_from_object(jayess_object_new()));
            jayess_object_set_value(request_object, "body", jayess_value_from_string(""));
            jayess_object_set_value(request_object, "timeout", jayess_object_get(object, "timeout") != NULL ? jayess_object_get(object, "timeout") : jayess_value_from_number(0));
            jayess_object_set_value(request_object, "maxRedirects", jayess_object_get(object, "maxRedirects") != NULL ? jayess_object_get(object, "maxRedirects") : jayess_value_from_number(5));
            jayess_object_set_value(request_object, "rejectUnauthorized", jayess_object_get(object, "rejectUnauthorized") != NULL ? jayess_object_get(object, "rejectUnauthorized") : jayess_value_from_bool(1));
            jayess_std_https_copy_tls_request_settings(request_object, object);
        }
        if (request_object != NULL && jayess_object_get(object, "headers") != NULL) {
            jayess_object_set_value(request_object, "headers", jayess_object_get(object, "headers"));
        }
        if (request_object != NULL && jayess_object_get(object, "version") != NULL) {
            jayess_object_set_value(request_object, "version", jayess_object_get(object, "version"));
        }
        if (request_object != NULL && jayess_object_get(object, "host") != NULL) {
            jayess_object_set_value(request_object, "host", jayess_object_get(object, "host"));
        }
        if (request_object != NULL && jayess_object_get(object, "port") != NULL) {
            jayess_object_set_value(request_object, "port", jayess_object_get(object, "port"));
        }
        if (request_object != NULL && jayess_object_get(object, "timeout") != NULL) {
            jayess_object_set_value(request_object, "timeout", jayess_object_get(object, "timeout"));
        }
        if (request_object != NULL && jayess_object_get(object, "maxRedirects") != NULL) {
            jayess_object_set_value(request_object, "maxRedirects", jayess_object_get(object, "maxRedirects"));
        }
        if (request_object != NULL) {
            jayess_std_https_copy_tls_request_settings(request_object, object);
        }
    }
    if (request_object == NULL) {
        return jayess_value_undefined();
    }
    jayess_object_set_value(request_object, "scheme", jayess_value_from_string("https"));
    if (jayess_value_as_object(jayess_object_get(request_object, "headers")) != NULL) {
        jayess_object *headers = jayess_value_as_object(jayess_object_get(request_object, "headers"));
        if (jayess_http_header_get_ci(headers, "Host") == NULL && jayess_object_get(request_object, "host") != NULL) {
            jayess_object_set_value(headers, "Host", jayess_object_get(request_object, "host"));
        }
        if (jayess_http_header_get_ci(headers, "Connection") == NULL) {
            jayess_object_set_value(headers, "Connection", jayess_value_from_string("close"));
        }
    }
    return jayess_https_request_stream_from_parts(request_object);
}

jayess_value *jayess_std_https_request_async(jayess_value *options) {
    jayess_value *promise = jayess_std_promise_pending();
    jayess_enqueue_http_request_task(promise, options, 0, 1, 0);
    return promise;
}

jayess_value *jayess_std_https_get_async(jayess_value *input) {
    jayess_object *object = jayess_value_as_object(input);
    if (object != NULL && jayess_https_get_uses_body_or_custom_method(object)) {
        return jayess_std_promise_reject(jayess_type_error_value("HTTPS request bodies and custom methods are not supported yet"));
    }
    jayess_value *promise = jayess_std_promise_pending();
    jayess_enqueue_http_request_task(promise, input, 1, 1, 0);
    return promise;
}

jayess_value *jayess_std_https_request_stream_async(jayess_value *options) {
    jayess_value *promise = jayess_std_promise_pending();
    jayess_enqueue_http_request_task(promise, options, 0, 1, 1);
    return promise;
}

jayess_value *jayess_std_https_get_stream_async(jayess_value *input) {
    jayess_object *object = jayess_value_as_object(input);
    if (object != NULL && jayess_https_get_uses_body_or_custom_method(object)) {
        return jayess_std_promise_reject(jayess_type_error_value("HTTPS request bodies and custom methods are not supported yet"));
    }
    jayess_value *promise = jayess_std_promise_pending();
    jayess_enqueue_http_request_task(promise, input, 1, 1, 1);
    return promise;
}

static int jayess_std_socket_runtime_ready(void) {
#ifdef _WIN32
    static int winsock_initialized = 0;
    if (!winsock_initialized) {
        WSADATA data;
        if (WSAStartup(MAKEWORD(2, 2), &data) != 0) {
            return 0;
        }
        winsock_initialized = 1;
    }
#endif
    return 1;
}

static int jayess_dns_address_family(const char *address) {
    unsigned char buffer[sizeof(struct in6_addr)];
    if (address == NULL || address[0] == '\0') {
        return 0;
    }
    if (inet_pton(AF_INET, address, buffer) == 1) {
        return 4;
    }
    if (inet_pton(AF_INET6, address, buffer) == 1) {
        return 6;
    }
    return 0;
}

static jayess_value *jayess_dns_make_record_value(const char *host, const char *address, int family) {
    jayess_object *record = jayess_object_new();
    jayess_object_set_value(record, "host", jayess_value_from_string(host != NULL ? host : ""));
    jayess_object_set_value(record, "address", jayess_value_from_string(address != NULL ? address : ""));
    jayess_object_set_value(record, "family", jayess_value_from_number((double)family));
    return jayess_value_from_object(record);
}

static jayess_object *jayess_dns_custom_hosts_object(void) {
    jayess_object *resolver = jayess_value_as_object(jayess_dns_custom_resolver);
    jayess_value *hosts_value;
    if (resolver == NULL) {
        return NULL;
    }
    hosts_value = jayess_object_get(resolver, "hosts");
    return jayess_value_as_object(hosts_value);
}

static jayess_object *jayess_dns_custom_reverse_object(void) {
    jayess_object *resolver = jayess_value_as_object(jayess_dns_custom_resolver);
    jayess_value *reverse_value;
    if (resolver == NULL) {
        return NULL;
    }
    reverse_value = jayess_object_get(resolver, "reverse");
    return jayess_value_as_object(reverse_value);
}

static jayess_value *jayess_dns_lookup_custom(jayess_value *host, int include_all) {
    jayess_object *hosts = jayess_dns_custom_hosts_object();
    char *host_text;
    jayess_value *mapped;
    if (hosts == NULL) {
        return NULL;
    }
    host_text = jayess_value_stringify(host);
    if (host_text == NULL || host_text[0] == '\0') {
        free(host_text);
        return NULL;
    }
    mapped = jayess_object_get(hosts, host_text);
    if (mapped == NULL || jayess_value_is_nullish(mapped)) {
        free(host_text);
        return NULL;
    }
    if (mapped->kind == JAYESS_VALUE_ARRAY && mapped->as.array_value != NULL) {
        jayess_array *records = jayess_array_new();
        int i;
        for (i = 0; i < mapped->as.array_value->count; i++) {
            char *address = jayess_value_stringify(mapped->as.array_value->values[i]);
            if (address == NULL || address[0] == '\0') {
                free(address);
                continue;
            }
            if (!include_all) {
                jayess_value *record = jayess_dns_make_record_value(host_text, address, jayess_dns_address_family(address));
                free(address);
                free(host_text);
                return record;
            }
            jayess_array_push_value(records, jayess_dns_make_record_value(host_text, address, jayess_dns_address_family(address)));
            free(address);
        }
        free(host_text);
        if (records->count == 0) {
            return NULL;
        }
        return jayess_value_from_array(records);
    }
    {
        char *address = jayess_value_stringify(mapped);
        jayess_value *record;
        if (address == NULL || address[0] == '\0') {
            free(address);
            free(host_text);
            return NULL;
        }
        record = jayess_dns_make_record_value(host_text, address, jayess_dns_address_family(address));
        free(address);
        free(host_text);
        if (include_all) {
            jayess_array *records = jayess_array_new();
            jayess_array_push_value(records, record);
            return jayess_value_from_array(records);
        }
        return record;
    }
}

static jayess_value *jayess_dns_reverse_custom(jayess_value *address) {
    jayess_object *reverse = jayess_dns_custom_reverse_object();
    char *address_text;
    jayess_value *mapped;
    char *host_text;
    if (reverse == NULL) {
        return NULL;
    }
    address_text = jayess_value_stringify(address);
    if (address_text == NULL || address_text[0] == '\0') {
        free(address_text);
        return NULL;
    }
    mapped = jayess_object_get(reverse, address_text);
    free(address_text);
    if (mapped == NULL || jayess_value_is_nullish(mapped)) {
        return NULL;
    }
    host_text = jayess_value_stringify(mapped);
    if (host_text == NULL || host_text[0] == '\0') {
        free(host_text);
        return NULL;
    }
    mapped = jayess_value_from_string(host_text);
    free(host_text);
    return mapped;
}

jayess_value *jayess_std_dns_lookup(jayess_value *host) {
    jayess_value *custom = jayess_dns_lookup_custom(host, 0);
    if (custom != NULL) {
        return custom;
    }
    char *host_text = jayess_value_stringify(host);
    const char *lookup_host = host_text != NULL ? host_text : "";
    struct addrinfo hints;
    struct addrinfo *results = NULL;
    struct addrinfo *entry;
    char address[INET6_ADDRSTRLEN];
    int family = 0;
    int status;
    jayess_object *object;

    if (lookup_host[0] == '\0') {
        free(host_text);
        return jayess_value_undefined();
    }

    if (!jayess_std_socket_runtime_ready()) {
        free(host_text);
        return jayess_value_undefined();
    }

    memset(&hints, 0, sizeof(hints));
    hints.ai_family = AF_UNSPEC;
    hints.ai_socktype = SOCK_STREAM;
    status = getaddrinfo(lookup_host, NULL, &hints, &results);
    if (status != 0 || results == NULL) {
        free(host_text);
        return jayess_value_undefined();
    }

    address[0] = '\0';
    for (entry = results; entry != NULL; entry = entry->ai_next) {
        void *addr = NULL;
        if (entry->ai_family == AF_INET) {
            struct sockaddr_in *ipv4 = (struct sockaddr_in *)entry->ai_addr;
            addr = &(ipv4->sin_addr);
            family = 4;
        } else if (entry->ai_family == AF_INET6) {
            struct sockaddr_in6 *ipv6 = (struct sockaddr_in6 *)entry->ai_addr;
            addr = &(ipv6->sin6_addr);
            family = 6;
        }
        if (addr != NULL && inet_ntop(entry->ai_family, addr, address, sizeof(address)) != NULL) {
            break;
        }
    }

    freeaddrinfo(results);
    if (address[0] == '\0') {
        free(host_text);
        return jayess_value_undefined();
    }

    object = jayess_object_new();
    jayess_object_set_value(object, "host", jayess_value_from_string(lookup_host));
    jayess_object_set_value(object, "address", jayess_value_from_string(address));
    jayess_object_set_value(object, "family", jayess_value_from_number((double)family));
    free(host_text);
    return jayess_value_from_object(object);
}

jayess_value *jayess_std_dns_lookup_all(jayess_value *host) {
    jayess_value *custom = jayess_dns_lookup_custom(host, 1);
    if (custom != NULL) {
        return custom;
    }
    char *host_text = jayess_value_stringify(host);
    const char *lookup_host = host_text != NULL ? host_text : "";
    struct addrinfo hints;
    struct addrinfo *results = NULL;
    struct addrinfo *entry;
    jayess_array *records;
    int status;

    if (lookup_host[0] == '\0') {
        free(host_text);
        return jayess_value_undefined();
    }

    if (!jayess_std_socket_runtime_ready()) {
        free(host_text);
        return jayess_value_undefined();
    }

    memset(&hints, 0, sizeof(hints));
    hints.ai_family = AF_UNSPEC;
    hints.ai_socktype = SOCK_STREAM;
    status = getaddrinfo(lookup_host, NULL, &hints, &results);
    if (status != 0 || results == NULL) {
        free(host_text);
        return jayess_value_undefined();
    }

    records = jayess_array_new();
    for (entry = results; entry != NULL; entry = entry->ai_next) {
        char address[INET6_ADDRSTRLEN];
        void *addr = NULL;
        int family = 0;
        jayess_object *record;
        if (entry->ai_family == AF_INET) {
            struct sockaddr_in *ipv4 = (struct sockaddr_in *)entry->ai_addr;
            addr = &(ipv4->sin_addr);
            family = 4;
        } else if (entry->ai_family == AF_INET6) {
            struct sockaddr_in6 *ipv6 = (struct sockaddr_in6 *)entry->ai_addr;
            addr = &(ipv6->sin6_addr);
            family = 6;
        }
        if (addr == NULL || inet_ntop(entry->ai_family, addr, address, sizeof(address)) == NULL) {
            continue;
        }
        record = jayess_object_new();
        jayess_object_set_value(record, "host", jayess_value_from_string(lookup_host));
        jayess_object_set_value(record, "address", jayess_value_from_string(address));
        jayess_object_set_value(record, "family", jayess_value_from_number((double)family));
        jayess_array_push_value(records, jayess_value_from_object(record));
    }

    freeaddrinfo(results);
    free(host_text);
    if (records->count == 0) {
        return jayess_value_undefined();
    }
    return jayess_value_from_array(records);
}

jayess_value *jayess_std_dns_reverse(jayess_value *address) {
    jayess_value *custom = jayess_dns_reverse_custom(address);
    if (custom != NULL) {
        return custom;
    }
    char *address_text = jayess_value_stringify(address);
    const char *lookup_address = address_text != NULL ? address_text : "";
    char host[NI_MAXHOST];
    unsigned char buffer[sizeof(struct in6_addr)];
    int status;

    if (lookup_address[0] == '\0' || !jayess_std_socket_runtime_ready()) {
        free(address_text);
        return jayess_value_undefined();
    }

    host[0] = '\0';
    if (inet_pton(AF_INET, lookup_address, buffer) == 1) {
        struct sockaddr_in addr;
        memset(&addr, 0, sizeof(addr));
        addr.sin_family = AF_INET;
        memcpy(&addr.sin_addr, buffer, sizeof(struct in_addr));
        status = getnameinfo((struct sockaddr *)&addr, sizeof(addr), host, sizeof(host), NULL, 0, 0);
    } else if (inet_pton(AF_INET6, lookup_address, buffer) == 1) {
        struct sockaddr_in6 addr;
        memset(&addr, 0, sizeof(addr));
        addr.sin6_family = AF_INET6;
        memcpy(&addr.sin6_addr, buffer, sizeof(struct in6_addr));
        status = getnameinfo((struct sockaddr *)&addr, sizeof(addr), host, sizeof(host), NULL, 0, 0);
    } else {
        free(address_text);
        return jayess_value_undefined();
    }

    free(address_text);
    if (status != 0 || host[0] == '\0') {
        return jayess_value_undefined();
    }
    return jayess_value_from_string(host);
}

jayess_value *jayess_std_dns_set_resolver(jayess_value *options) {
    if (!jayess_value_is_nullish(options) && jayess_value_as_object(options) == NULL) {
        jayess_throw(jayess_type_error_value("dns.setResolver expects an object or null"));
        return jayess_value_undefined();
    }
    jayess_dns_custom_resolver = jayess_value_is_nullish(options) ? NULL : options;
    return jayess_value_from_bool(1);
}

jayess_value *jayess_std_dns_clear_resolver(void) {
    jayess_dns_custom_resolver = NULL;
    return jayess_value_from_bool(1);
}

jayess_value *jayess_std_net_is_ip(jayess_value *input) {
    char *text = jayess_value_stringify(input);
    unsigned char buffer[sizeof(struct in6_addr)];
    int family = 0;

    if (text != NULL && inet_pton(AF_INET, text, buffer) == 1) {
        family = 4;
    } else if (text != NULL && inet_pton(AF_INET6, text, buffer) == 1) {
        family = 6;
    }

    free(text);
    return jayess_value_from_number((double)family);
}

jayess_value *jayess_std_net_create_datagram_socket(jayess_value *options) {
    jayess_object *object_options = jayess_value_as_object(options);
    jayess_value *host_value = object_options != NULL ? jayess_object_get(object_options, "host") : NULL;
    jayess_value *port_value = object_options != NULL ? jayess_object_get(object_options, "port") : NULL;
    jayess_value *type_value = object_options != NULL ? jayess_object_get(object_options, "type") : NULL;
    jayess_value *timeout_value = object_options != NULL ? jayess_object_get(object_options, "timeout") : NULL;
    jayess_value *broadcast_value = object_options != NULL ? jayess_object_get(object_options, "broadcast") : NULL;
    char *type_text = jayess_value_stringify(type_value);
    char *host_text = jayess_value_stringify(host_value);
    int port = (int)jayess_value_to_number(port_value);
    int timeout = (int)jayess_value_to_number(timeout_value);
    int enable_broadcast = jayess_value_as_bool(broadcast_value);
    char port_text[32];
    struct addrinfo hints;
    struct addrinfo *results = NULL;
    struct addrinfo *entry;
    jayess_socket_handle handle = JAYESS_INVALID_SOCKET;
    int family = 4;
    int status;
    if (type_text != NULL && type_text[0] != '\0') {
        if (strcmp(type_text, "udp6") == 0) {
            family = 6;
        } else if (strcmp(type_text, "udp4") != 0) {
            free(type_text);
            free(host_text);
            return jayess_value_undefined();
        }
    }
    if ((host_text == NULL || host_text[0] == '\0')) {
        free(host_text);
        host_text = jayess_strdup(family == 6 ? "::1" : "127.0.0.1");
    }
    if (port < 0 || !jayess_std_socket_runtime_ready()) {
        free(type_text);
        free(host_text);
        return jayess_value_undefined();
    }
    snprintf(port_text, sizeof(port_text), "%d", port);
    memset(&hints, 0, sizeof(hints));
    hints.ai_family = family == 6 ? AF_INET6 : AF_INET;
    hints.ai_socktype = SOCK_DGRAM;
    hints.ai_flags = AI_PASSIVE;
    status = getaddrinfo(host_text, port_text, &hints, &results);
    if (status != 0 || results == NULL) {
        free(type_text);
        free(host_text);
        return jayess_value_undefined();
    }
    for (entry = results; entry != NULL; entry = entry->ai_next) {
        handle = socket(entry->ai_family, entry->ai_socktype, entry->ai_protocol);
        if (handle == JAYESS_INVALID_SOCKET) {
            continue;
        }
        if (bind(handle, entry->ai_addr, (int)entry->ai_addrlen) == 0) {
            family = entry->ai_family == AF_INET6 ? 6 : 4;
            break;
        }
        jayess_std_socket_close_handle(handle);
        handle = JAYESS_INVALID_SOCKET;
    }
    freeaddrinfo(results);
    free(type_text);
    free(host_text);
    if (handle == JAYESS_INVALID_SOCKET) {
        return jayess_value_undefined();
    }
    if (timeout > 0 && !jayess_std_socket_configure_timeout(handle, timeout)) {
        jayess_std_socket_close_handle(handle);
        return jayess_value_undefined();
    }
    {
        jayess_value *result = jayess_std_datagram_socket_value_from_handle(handle);
        if (result == NULL || result->kind != JAYESS_VALUE_OBJECT || result->as.object_value == NULL) {
            jayess_std_socket_close_handle(handle);
            return jayess_value_undefined();
        }
        jayess_object_set_value(result->as.object_value, "timeout", jayess_value_from_number((double)timeout));
        jayess_object_set_value(result->as.object_value, "localFamily", jayess_value_from_number((double)family));
        jayess_std_socket_set_local_endpoint(result, handle);
        if (enable_broadcast) {
            jayess_std_datagram_socket_set_broadcast_method(result, jayess_value_from_bool(1));
        }
        return result;
    }
}

jayess_value *jayess_std_net_connect(jayess_value *options) {
    jayess_object *object_options = jayess_value_as_object(options);
    jayess_value *host_value = object_options != NULL ? jayess_object_get(object_options, "host") : NULL;
    jayess_value *port_value = object_options != NULL ? jayess_object_get(object_options, "port") : NULL;
    char *host_text = jayess_value_stringify(host_value);
    int port = (int)jayess_value_to_number(port_value);
    char port_text[32];
    struct addrinfo hints;
    struct addrinfo *results = NULL;
    struct addrinfo *entry;
    jayess_socket_handle handle = JAYESS_INVALID_SOCKET;
    int family = 0;
    int status;

    if (host_text == NULL || host_text[0] == '\0' || port <= 0 || !jayess_std_socket_runtime_ready()) {
        free(host_text);
        return jayess_value_undefined();
    }

    snprintf(port_text, sizeof(port_text), "%d", port);
    memset(&hints, 0, sizeof(hints));
    hints.ai_family = AF_UNSPEC;
    hints.ai_socktype = SOCK_STREAM;
    status = getaddrinfo(host_text, port_text, &hints, &results);
    if (status != 0 || results == NULL) {
        free(host_text);
        return jayess_value_undefined();
    }

    for (entry = results; entry != NULL; entry = entry->ai_next) {
        handle = socket(entry->ai_family, entry->ai_socktype, entry->ai_protocol);
        if (handle == JAYESS_INVALID_SOCKET) {
            continue;
        }
        if (connect(handle, entry->ai_addr, (int)entry->ai_addrlen) == 0) {
            family = entry->ai_family == AF_INET6 ? 6 : 4;
            break;
        }
        jayess_std_socket_close_handle(handle);
        handle = JAYESS_INVALID_SOCKET;
    }

    freeaddrinfo(results);
    if (handle == JAYESS_INVALID_SOCKET) {
        free(host_text);
        return jayess_value_undefined();
    }

    {
        jayess_value *result = jayess_std_socket_value_from_handle(handle, host_text, port);
        jayess_std_socket_set_remote_family(result, family);
        jayess_std_socket_set_local_endpoint(result, handle);
        free(host_text);
        return result;
    }
}

jayess_value *jayess_std_net_listen(jayess_value *options) {
    jayess_object *object_options = jayess_value_as_object(options);
    jayess_value *host_value = object_options != NULL ? jayess_object_get(object_options, "host") : NULL;
    jayess_value *port_value = object_options != NULL ? jayess_object_get(object_options, "port") : NULL;
    char *host_text = jayess_value_stringify(host_value);
    int port = (int)jayess_value_to_number(port_value);
    char port_text[32];
    struct addrinfo hints;
    struct addrinfo *results = NULL;
    struct addrinfo *entry;
    jayess_socket_handle handle = JAYESS_INVALID_SOCKET;
    int family = 0;
    int status;
    jayess_object *server_object;
    int yes = 1;

    if (host_text == NULL || host_text[0] == '\0' || port < 0 || !jayess_std_socket_runtime_ready()) {
        free(host_text);
        return jayess_value_undefined();
    }

    snprintf(port_text, sizeof(port_text), "%d", port);
    memset(&hints, 0, sizeof(hints));
    hints.ai_family = AF_UNSPEC;
    hints.ai_socktype = SOCK_STREAM;
    hints.ai_flags = AI_PASSIVE;
    status = getaddrinfo(host_text, port_text, &hints, &results);
    if (status != 0 || results == NULL) {
        free(host_text);
        return jayess_value_undefined();
    }

    for (entry = results; entry != NULL; entry = entry->ai_next) {
        handle = socket(entry->ai_family, entry->ai_socktype, entry->ai_protocol);
        if (handle == JAYESS_INVALID_SOCKET) {
            continue;
        }
#ifdef _WIN32
        setsockopt(handle, SOL_SOCKET, SO_REUSEADDR, (const char *)&yes, sizeof(yes));
#else
        setsockopt(handle, SOL_SOCKET, SO_REUSEADDR, &yes, sizeof(yes));
#endif
        if (bind(handle, entry->ai_addr, (int)entry->ai_addrlen) == 0 && listen(handle, 16) == 0) {
            family = entry->ai_family == AF_INET6 ? 6 : 4;
            break;
        }
        jayess_std_socket_close_handle(handle);
        handle = JAYESS_INVALID_SOCKET;
    }
    freeaddrinfo(results);
    if (handle == JAYESS_INVALID_SOCKET) {
        free(host_text);
        return jayess_value_undefined();
    }

    if (port == 0) {
        struct sockaddr_storage local_addr;
#ifdef _WIN32
        int local_len = sizeof(local_addr);
#else
        socklen_t local_len = sizeof(local_addr);
#endif
        memset(&local_addr, 0, sizeof(local_addr));
        if (getsockname(handle, (struct sockaddr *)&local_addr, &local_len) == 0) {
            if (local_addr.ss_family == AF_INET) {
                port = ntohs(((struct sockaddr_in *)&local_addr)->sin_port);
            } else if (local_addr.ss_family == AF_INET6) {
                port = ntohs(((struct sockaddr_in6 *)&local_addr)->sin6_port);
            }
        }
    }

    server_object = jayess_object_new();
    if (server_object == NULL) {
        jayess_std_socket_close_handle(handle);
        free(host_text);
        return jayess_value_from_object(NULL);
    }
    server_object->socket_handle = handle;
    jayess_object_set_value(server_object, "__jayess_std_kind", jayess_value_from_string("Server"));
    jayess_object_set_value(server_object, "listening", jayess_value_from_bool(1));
    jayess_object_set_value(server_object, "closed", jayess_value_from_bool(0));
    jayess_object_set_value(server_object, "host", jayess_value_from_string(host_text));
    jayess_object_set_value(server_object, "port", jayess_value_from_number((double)port));
    jayess_object_set_value(server_object, "family", jayess_value_from_number((double)family));
    jayess_object_set_value(server_object, "timeout", jayess_value_from_number(0));
    jayess_object_set_value(server_object, "connectionsAccepted", jayess_value_from_number(0));
    jayess_object_set_value(server_object, "errored", jayess_value_from_bool(0));
    free(host_text);
    return jayess_value_from_object(server_object);
}
